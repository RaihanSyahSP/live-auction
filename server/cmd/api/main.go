package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"live-auction/server/internal/app"
	"live-auction/server/internal/auth"
	"live-auction/server/internal/config"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/repository"
	auctionrepo "live-auction/server/internal/repository/auction"
	bidrepo "live-auction/server/internal/repository/bid"
	refreshrepo "live-auction/server/internal/repository/refresh_token"
	userrepo "live-auction/server/internal/repository/user"
	"live-auction/server/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("load .env", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseCtx, cancelDB := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelDB()

	pool, err := app.NewDB(databaseCtx, cfg)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	userRepository := userrepo.NewUserRepositoryImpl()
	refreshRepository := refreshrepo.NewRefreshTokenRepositoryImpl()
	txManager := repository.NewTxManager(pool)
	bidRepository := bidrepo.NewBidRepositoryImpl()
	auctionRepository := auctionrepo.NewAuctionRepositoryImpl()

	accessTokens := auth.NewAccessTokens(
		[]byte(cfg.JWTSecret),
		"live-auction",
		"live-auction-web",
		cfg.AccessTokenTTL,
	)
	cookies := auth.NewCookies(
		cfg.CookieSecure,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)

	authService := service.NewAuthService(
		userRepository,
		refreshRepository,
		txManager,
		accessTokens,
		time.Now,
		cfg.RefreshTokenTTL,
		cfg.RefreshGracePeriod,
	)

	auctionService := service.NewAuctionService(
		auctionRepository,
		bidRepository,
		txManager,
		time.Now,
	)

	bidService := service.NewBidService(
		auctionRepository,
		bidRepository,
		txManager,
		time.Now,
	)

	rooms := realtime.NewRoomManager()

	scheduler := app.NewScheduler(auctionService, rooms, time.Second, time.Now)

	startupCtx, cancelStartup := context.WithTimeout(ctx, 10*time.Second)
	if err := scheduler.CloseDue(startupCtx); err != nil {
		cancelStartup()
		slog.Error("close due auctions on startup", "error", err)
		os.Exit(1)
	}
	cancelStartup()

	wsController := controller.NewWebSocketController(
		rooms,
		auctionService,
		accessTokens,
		cfg.AppOrigin,
	)
	auctionController := controller.NewAuctionController(auctionService)
	adminController := controller.NewAdminController(auctionService, rooms)
	authController := controller.NewAuthController(authService, cookies)
	bidController := controller.NewBidController(bidService, rooms)

	loginLimiter := middleware.NewRateLimiter(
		rate.Every(6*time.Second), // 10 per menit
		10,
	)
	refreshLimiter := middleware.NewRateLimiter(
		rate.Every(2*time.Second), // 30 per menit
		30,
	)

	bidLimiter := middleware.NewRateLimiter(rate.Limit(3), 5)

	handler := app.NewRouter(app.RouterDeps{
		Health:         controller.NewHealthController(),
		Auth:           authController,
		AccessTokens:   accessTokens,
		AllowedOrigin:  cfg.AppOrigin,
		LoginLimiter:   loginLimiter,
		RefreshLimiter: refreshLimiter,
		Auction:        auctionController,
		Admin:          adminController,
		WebSocket:      wsController,
		Bid:            bidController,
		BidLimiter:     bidLimiter,
	})
	server := app.NewServer(cfg, handler)

	go scheduler.Run(ctx)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
