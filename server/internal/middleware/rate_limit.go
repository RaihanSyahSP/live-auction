package middleware

import (
	"live-auction/server/internal/exception"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
	visitors map[string]*rate.Limiter
}

func NewRateLimiter(limit rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		burst:    burst,
		visitors: make(map[string]*rate.Limiter),
	}
}

func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	limiter := l.visitors[key]
	if limiter == nil {
		limiter = rate.NewLimiter(l.limit, l.burst)
		l.visitors[key] = limiter
	}
	l.mu.Unlock()

	return limiter.Allow()
}

func RateLimit(
	limiter *RateLimiter,
	keyFunc func(*http.Request) string,
	next http.Handler,
) http.Handler {
	if limiter == nil {
		return next
	}
	if keyFunc == nil {
		keyFunc = ClientIP
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(keyFunc(r)) {
			exception.WriteError(w, exception.ErrRateLimited)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
