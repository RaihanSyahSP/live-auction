package middleware

import (
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"net/http"
)

func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFrom(r.Context())
		if !ok || principal.Role != domain.RoleAdmin {
			exception.WriteError(w, exception.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
