package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"live-auction/server/internal/app"
	"live-auction/server/internal/config"
	"live-auction/server/internal/seed"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := app.NewDB(ctx, cfg)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := seed.Run(ctx, pool, seed.Config{
		AdminEmail:     cfg.SeedAdminEmail,
		AdminPassword:  cfg.SeedAdminPassword,
		BidderPassword: cfg.SeedBidderPassword,
	}); err != nil {
		slog.Error("run seed", "error", err)
		os.Exit(1)
	}

	slog.Info("seed complete")
}
