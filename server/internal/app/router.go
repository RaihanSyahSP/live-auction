package app

import (
	"net/http"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/middleware"
)

type RouterDeps struct {
	Health         *controller.HealthController
	Auth           *controller.AuthController
	AccessTokens   *auth.AccessTokens
	AllowedOrigin  string
	LoginLimiter   *middleware.RateLimiter
	RefreshLimiter *middleware.RateLimiter
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	if deps.Health != nil {
		mux.HandleFunc("GET /healthz", deps.Health.Health)
	}

	if deps.Auth == nil {
		return mux
	}

	register := http.HandlerFunc(deps.Auth.Register)
	login := http.HandlerFunc(deps.Auth.Login)
	refresh := http.HandlerFunc(deps.Auth.Refresh)
	logout := http.HandlerFunc(deps.Auth.Logout)
	me := middleware.Auth(
		deps.AccessTokens,
		http.HandlerFunc(deps.Auth.Me),
	)

	mux.Handle(
		"POST /api/auth/register",
		middleware.Origin(
			deps.AllowedOrigin,
			middleware.RateLimit(deps.LoginLimiter, middleware.ClientIP, register),
		),
	)

	mux.Handle(
		"POST /api/auth/login",
		middleware.Origin(
			deps.AllowedOrigin,
			middleware.RateLimit(deps.LoginLimiter, middleware.ClientIP, login),
		),
	)
	mux.Handle(
		"POST /api/auth/refresh",
		middleware.Origin(
			deps.AllowedOrigin,
			middleware.RateLimit(deps.RefreshLimiter, middleware.ClientIP, refresh),
		),
	)
	mux.Handle(
		"POST /api/auth/logout",
		middleware.Origin(deps.AllowedOrigin, logout),
	)
	mux.Handle("GET /api/auth/me", me)

	return mux
}
