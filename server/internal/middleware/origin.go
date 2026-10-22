package middleware

import (
	"net/http"
	"strings"

	"live-auction/server/internal/exception"
)

func Origin(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && origin != allowedOrigin {
			exception.WriteError(w, exception.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
