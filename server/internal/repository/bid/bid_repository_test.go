package bid_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	bidrepo "live-auction/server/internal/repository/bid"
)

func newBidRepo(t *testing.T) (*bidrepo.BidRepositoryImpl, pgx.Tx) {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	return bidrepo.NewBidRepositoryImpl(), tx
}

func createBidUser(t *testing.T, tx pgx.Tx, displayName string) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(
		context.Background(),
		`insert into users (email, password_hash, display_name, role)
		 values ($1, $2, $3, 'user')
		 returning id`,
		uuid.NewString()+"@example.com",
		"hash",
		displayName,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	return id
}

func createBidAuction(t *testing.T, tx pgx.Tx, createdBy uuid.UUID) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(
		context.Background(),
		`insert into auctions (
			created_by, lot, title, description, condition,
			starting_price, min_increment, duration_seconds, current_price,
			status, starts_at, ends_at
		) values ($1, $2, 'Test auction', 'Test description', 'Used',
			1000, 100, 600, 1000, 'live', now(), now() + interval '10 minutes')
		returning id`,
		createdBy,
		uuid.NewString(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("create auction fixture: %v", err)
	}
	return id
}

func TestBidRepositoryInsertAndFind(t *testing.T) {
	repo, tx := newBidRepo(t)
	ctx := context.Background()
	userID := createBidUser(t, tx, "Raihan")
	auctionID := createBidAuction(t, tx, userID)
	key := uuid.New()

	inserted, err := repo.Insert(ctx, tx, domain.Bid{
		AuctionID:      auctionID,
		UserID:         userID,
		Amount:         1500,
		IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	if inserted.ID == uuid.Nil {
		t.Error("Insert() returned nil ID")
	}
	if inserted.CreatedAt.IsZero() {
		t.Error("Insert() returned zero CreatedAt")
	}
	if inserted.DisplayName != "Raihan" {
		t.Errorf("Insert() DisplayName = %q, want Raihan", inserted.DisplayName)
	}

	found, err := repo.FindByIdempotencyKey(ctx, tx, auctionID, userID, key)
	if err != nil {
		t.Fatalf("FindByIdempotencyKey() error = %v", err)
	}
	if found.ID != inserted.ID {
		t.Errorf("FindByIdempotencyKey() ID = %v, want %v", found.ID, inserted.ID)
	}
	if found.DisplayName != "Raihan" {
		t.Errorf("FindByIdempotencyKey() DisplayName = %q, want Raihan", found.DisplayName)
	}
}

func TestBidRepositoryRejectsDuplicateIdempotencyKey(t *testing.T) {
	repo, tx := newBidRepo(t)
	ctx := context.Background()
	userID := createBidUser(t, tx, "Raihan")
	auctionID := createBidAuction(t, tx, userID)
	key := uuid.New()

	if _, err := repo.Insert(ctx, tx, domain.Bid{
		AuctionID:      auctionID,
		UserID:         userID,
		Amount:         1500,
		IdempotencyKey: key,
	}); err != nil {
		t.Fatalf("first Insert() error = %v", err)
	}

	_, err := repo.Insert(ctx, tx, domain.Bid{
		AuctionID:      auctionID,
		UserID:         userID,
		Amount:         1600,
		IdempotencyKey: key,
	})
	if !errors.Is(err, exception.ErrBidAlreadyExists) {
		t.Fatalf("second Insert() error = %v, want %v", err, exception.ErrBidAlreadyExists)
	}
}

func TestBidRepositoryAllowsSameIdempotencyKeyOnDifferentAuction(t *testing.T) {
	repo, tx := newBidRepo(t)
	ctx := context.Background()
	userID := createBidUser(t, tx, "Raihan")
	firstAuctionID := createBidAuction(t, tx, userID)

	var secondAuctionID uuid.UUID
	err := tx.QueryRow(
		ctx,
		`insert into auctions (
			created_by, lot, title, description, condition,
			starting_price, min_increment, duration_seconds, current_price, status
		) values ($1, $2, 'Draft auction', 'Test description', 'Used',
			1000, 100, 600, 1000, 'draft')
		returning id`,
		userID,
		uuid.NewString(),
	).Scan(&secondAuctionID)
	if err != nil {
		t.Fatalf("create draft auction fixture: %v", err)
	}

	key := uuid.New()
	if _, err := repo.Insert(ctx, tx, domain.Bid{
		AuctionID:      firstAuctionID,
		UserID:         userID,
		Amount:         1500,
		IdempotencyKey: key,
	}); err != nil {
		t.Fatalf("first Insert() error = %v", err)
	}

	if _, err := repo.Insert(ctx, tx, domain.Bid{
		AuctionID:      secondAuctionID,
		UserID:         userID,
		Amount:         1500,
		IdempotencyKey: key,
	}); err != nil {
		t.Fatalf("second Insert() error = %v", err)
	}
}

func TestBidRepositoryFindByIdempotencyKeyNotFound(t *testing.T) {
	repo, tx := newBidRepo(t)

	_, err := repo.FindByIdempotencyKey(context.Background(), tx, uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, exception.ErrBidNotFound) {
		t.Fatalf("FindByIdempotencyKey() error = %v, want %v", err, exception.ErrBidNotFound)
	}
}

func TestBidRepositoryListByAuctionWithPagination(t *testing.T) {
	repo, tx := newBidRepo(t)
	ctx := context.Background()
	userID := createBidUser(t, tx, "Raihan")
	auctionID := createBidAuction(t, tx, userID)

	ids := []uuid.UUID{
		uuid.MustParse("018f0000-0000-7000-8000-000000000001"),
		uuid.MustParse("018f0000-0000-7000-8000-000000000002"),
		uuid.MustParse("018f0000-0000-7000-8000-000000000003"),
	}
	for i, id := range ids {
		_, err := tx.Exec(
			ctx,
			`insert into bids (id, auction_id, user_id, amount, idempotency_key)
			 values ($1, $2, $3, $4, $5)`,
			id,
			auctionID,
			userID,
			1000+i,
			uuid.New(),
		)
		if err != nil {
			t.Fatalf("insert bid fixture: %v", err)
		}
	}

	firstPage, err := repo.ListByAuction(ctx, tx, auctionID, nil, 2)
	if err != nil {
		t.Fatalf("ListByAuction() error = %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("first page length = %d, want 2", len(firstPage))
	}
	if firstPage[0].ID != ids[2] || firstPage[1].ID != ids[1] {
		t.Errorf("first page IDs = %v, want [%v %v]", []uuid.UUID{firstPage[0].ID, firstPage[1].ID}, ids[2], ids[1])
	}
	if firstPage[0].DisplayName != "Raihan" {
		t.Errorf("DisplayName = %q, want Raihan", firstPage[0].DisplayName)
	}

	secondPage, err := repo.ListByAuction(ctx, tx, auctionID, &firstPage[1].ID, 2)
	if err != nil {
		t.Fatalf("ListByAuction() second page error = %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("second page length = %d, want 1", len(secondPage))
	}
	if secondPage[0].ID != ids[0] {
		t.Errorf("second page ID = %v, want %v", secondPage[0].ID, ids[0])
	}
}
