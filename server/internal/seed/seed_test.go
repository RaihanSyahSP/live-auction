package seed

import (
	"context"
	"testing"

	"live-auction/server/internal/helper"
)

func TestRunTxIsIdempotent(t *testing.T) {
	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	ctx := context.Background()

	cfg := Config{
		AdminEmail:     "admin-seed-test@example.com",
		AdminPassword:  "admin-password",
		BidderPassword: "bidder-password",
	}

	if err := runTx(ctx, tx, cfg); err != nil {
		t.Fatalf("first runTx() error = %v", err)
	}
	if err := runTx(ctx, tx, cfg); err != nil {
		t.Fatalf("second runTx() error = %v", err)
	}

	var userCount int
	if err := tx.QueryRow(
		ctx,
		`select count(*) from users
		 where lower(email) in (lower($1), lower($2))`,
		cfg.AdminEmail,
		BidderEmail,
	).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 2 {
		t.Errorf("seed users = %d, want 2", userCount)
	}

	var auctionCount int
	if err := tx.QueryRow(
		ctx,
		`select count(*) from auctions where lot in ($1, $2)`,
		LiveLot,
		DraftLot,
	).Scan(&auctionCount); err != nil {
		t.Fatalf("count auctions: %v", err)
	}
	if auctionCount != 2 {
		t.Errorf("seed auctions = %d, want 2", auctionCount)
	}

	var liveCount int
	if err := tx.QueryRow(
		ctx,
		`select count(*) from auctions where status = 'live'`,
	).Scan(&liveCount); err != nil {
		t.Fatalf("count live auctions: %v", err)
	}
	if liveCount > 1 {
		t.Errorf("live auctions = %d, want at most 1", liveCount)
	}
}

func TestRunRejectsMissingCredentials(t *testing.T) {
	if err := Run(context.Background(), nil, Config{}); err == nil {
		t.Fatal("Run() error = nil, want validation error")
	}
}
