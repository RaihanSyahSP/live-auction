package auction_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	auctionrepo "live-auction/server/internal/repository/auction"
)

func newAuctionRepo(t *testing.T) (*auctionrepo.AuctionRepositoryImpl, pgx.Tx) {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	return auctionrepo.NewAuctionRepositoryImpl(), tx
}

func createAuctionUser(t *testing.T, tx pgx.Tx) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(
		context.Background(),
		`insert into users (email, password_hash, display_name, role)
		 values ($1, 'hash', 'Auction User', 'user')
		 returning id`,
		uuid.NewString()+"@example.com",
	).Scan(&id)
	if err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	return id
}

func TestAuctionRepositoryCreateAndFind(t *testing.T) {
	repo, tx := newAuctionRepo(t)
	ctx := context.Background()
	userID := createAuctionUser(t, tx)

	created, err := repo.Create(ctx, tx, domain.Auction{
		CreatedBy:       userID,
		Lot:             "LOT-" + uuid.NewString()[:8],
		Title:           "Fujifilm X-T2",
		Description:     "Mirrorless APS-C",
		Condition:       "Used",
		Includes:        []string{"Body", "Charger"},
		Specs:           []domain.Spec{{Label: "Sensor", Value: "24 MP"}},
		ImageURLs:       []string{},
		StartingPrice:   5_000_000,
		MinIncrement:    100_000,
		DurationSeconds: 600,
		CurrentPrice:    5_000_000,
		Status:          domain.AuctionStatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(ctx, tx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
	if len(found.Specs) != 1 || found.Specs[0].Label != "Sensor" {
		t.Errorf("Specs = %+v", found.Specs)
	}
}

func TestAuctionRepositoryFindByIDNotFound(t *testing.T) {
	repo, tx := newAuctionRepo(t)

	_, err := repo.FindByID(context.Background(), tx, uuid.New())
	if !errors.Is(err, exception.ErrAuctionNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, exception.ErrAuctionNotFound)
	}
}

func TestAuctionRepositoryListDue(t *testing.T) {
	repo, tx := newAuctionRepo(t)
	ctx := context.Background()
	userID := createAuctionUser(t, tx)
	now := time.Now().UTC()
	endsAt := now.Add(-time.Minute)

	_, err := repo.Create(ctx, tx, domain.Auction{
		CreatedBy:       userID,
		Lot:             "LOT-" + uuid.NewString()[:8],
		Title:           "Due auction",
		Description:     "Test",
		Condition:       "Used",
		StartingPrice:   1000,
		MinIncrement:    100,
		DurationSeconds: 60,
		CurrentPrice:    1000,
		Status:          domain.AuctionStatusLive,
		StartsAt:        ptrTime(now.Add(-time.Hour)),
		EndsAt:          &endsAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	due, err := repo.ListDue(ctx, tx, now)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("ListDue() length = %d, want 1", len(due))
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
