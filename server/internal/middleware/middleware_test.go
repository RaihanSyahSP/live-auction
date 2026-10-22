package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/domain"
)

const middlewareTestSecret = "01234567890123456789012345678901"

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error.Code
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte(middlewareTestSecret),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)
	userID := uuid.New()
	raw, err := tokens.Issue(userID, domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	var principal auth.Principal
	handler := middleware.Auth(tokens, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, _ = middleware.PrincipalFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.AccessCookieName, Value: raw})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if principal.UserID != userID {
		t.Errorf("principal UserID = %v, want %v", principal.UserID, userID)
	}
	if principal.Role != domain.RoleUser {
		t.Errorf("principal Role = %v, want %v", principal.Role, domain.RoleUser)
	}
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte(middlewareTestSecret),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)

	handler := middleware.Auth(tokens, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "UNAUTHENTICATED" {
		t.Errorf("error code = %q, want UNAUTHENTICATED", code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte(middlewareTestSecret),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)

	handler := middleware.Auth(tokens, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.AccessCookieName, Value: "not-a-jwt"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "TOKEN_INVALID" {
		t.Errorf("error code = %q, want TOKEN_INVALID", code)
	}
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte(middlewareTestSecret),
		"live-auction",
		"live-auction-web",
		-time.Minute,
	)
	raw, err := tokens.Issue(uuid.New(), domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	handler := middleware.Auth(tokens, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.AccessCookieName, Value: raw})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "TOKEN_EXPIRED" {
		t.Errorf("error code = %q, want TOKEN_EXPIRED", code)
	}
}

func TestAdminMiddleware(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte(middlewareTestSecret),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)
	userID := uuid.New()

	adminToken, err := tokens.Issue(userID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("Issue(admin) error = %v", err)
	}
	userToken, err := tokens.Issue(userID, domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue(user) error = %v", err)
	}

	handler := middleware.Auth(tokens, middleware.Admin(okHandler()))

	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "admin", raw: adminToken, want: http.StatusOK},
		{name: "user", raw: userToken, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/test", nil)
			req.AddCookie(&http.Cookie{Name: auth.AccessCookieName, Value: tt.raw})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestOriginMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		method string
		origin string
		want   int
	}{
		{name: "allowed origin", method: http.MethodPost, origin: "http://localhost:5173", want: http.StatusOK},
		{name: "empty origin", method: http.MethodPost, want: http.StatusOK},
		{name: "wrong origin", method: http.MethodPost, origin: "http://evil.example", want: http.StatusForbidden},
		{name: "GET wrong origin", method: http.MethodGet, origin: "http://evil.example", want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.Origin("http://localhost:5173", okHandler())
			req := httptest.NewRequest(tt.method, "/api/auth/login", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := middleware.NewRateLimiter(rate.Every(time.Hour), 1)
	handler := middleware.RateLimit(limiter, func(r *http.Request) string {
		return "same-client"
	}, okHandler())

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if code := errorCode(t, second); code != "RATE_LIMITED" {
		t.Errorf("error code = %q, want RATE_LIMITED", code)
	}
}
