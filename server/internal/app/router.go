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
	Auction        *controller.AuctionController
	Admin          *controller.AdminController
	Bid            *controller.BidController
	BidLimiter     *middleware.RateLimiter
	WebSocket      *controller.WebSocketController
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	if deps.Health != nil {
		mux.HandleFunc("GET /healthz", deps.Health.Health)
	}

	if deps.Auth == nil {
		return mux
	}

	if deps.WebSocket != nil {
		mux.HandleFunc("GET /ws/auctions/{id}", deps.WebSocket.ServeWS)
	}

	if deps.Auction != nil {
		mux.HandleFunc("GET /api/auctions/current", deps.Auction.Current)
		mux.HandleFunc("GET /api/auctions/{id}", deps.Auction.Detail)
		mux.HandleFunc("GET /api/auctions/{id}/bids", deps.Auction.Bids)
		mux.HandleFunc("GET /api/results", deps.Auction.Results)
	}

	if deps.Auth != nil && deps.Admin != nil {
		adminOnly := func(h http.Handler) http.Handler {
			return middleware.Auth(
				deps.AccessTokens,
				middleware.Admin(h),
			)
		}
		adminStateChange := func(h http.Handler) http.Handler {
			return middleware.Origin(
				deps.AllowedOrigin,
				adminOnly(h),
			)
		}

		mux.Handle("GET /api/admin/auctions", adminOnly(http.HandlerFunc(deps.Admin.List)))
		mux.Handle("POST /api/admin/auctions", adminStateChange(http.HandlerFunc(deps.Admin.Create)))
		mux.Handle("POST /api/admin/auctions/{id}/start", adminStateChange(http.HandlerFunc(deps.Admin.Start)))
		mux.Handle("POST /api/admin/auctions/{id}/end", adminStateChange(http.HandlerFunc(deps.Admin.End)))
		mux.Handle("DELETE /api/admin/auctions/{id}", adminStateChange(http.HandlerFunc(deps.Admin.Delete)))
	}

	if deps.Bid != nil && deps.AccessTokens != nil {
		bidHandler := middleware.RateLimit(
			deps.BidLimiter,
			middleware.UserID,
			http.HandlerFunc(deps.Bid.PlaceBid),
		)
		mux.Handle(
			"POST /api/auctions/{id}/bids",
			middleware.Origin(
				deps.AllowedOrigin,
				middleware.Auth(deps.AccessTokens, bidHandler),
			),
		)
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
