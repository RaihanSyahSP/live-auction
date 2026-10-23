package middleware

import (
	"context"
	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
	"net/http"
)

type principalContextKey struct{}

func Auth(tokens *auth.AccessTokens, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.AccessCookieName)
		if err != nil {
			exception.WriteError(w, exception.ErrUnauthenticated)
			return
		}

		principal, err := tokens.Parse(cookie.Value)
		if err != nil {
			exception.WriteError(w, err)
			return
		}

		ctx := WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithPrincipal(ctx context.Context, principal auth.Principal) context.Context {
	ctx = context.WithValue(ctx, principalContextKey{}, principal)

	if meta := logMetaFrom(ctx); meta != nil {
		userID := principal.UserID
		meta.UserID = &userID
	}

	return ctx
}

func PrincipalFrom(ctx context.Context) (auth.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(auth.Principal)
	return principal, ok
}
