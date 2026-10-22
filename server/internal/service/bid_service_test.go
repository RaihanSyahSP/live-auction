package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/bid_request"
	"live-auction/server/internal/model/web/bid_response"
	"live-auction/server/internal/repository"
	auctionrepo "live-auction/server/internal/repository/auction"
	bidrepo "live-auction/server/internal/repository/bid"
	"live-auction/server/internal/service"
)

type bidFixture struct {
	svc      service.BidService
	auctions auctionrepo.AuctionRepository
	bids     bidrepo.BidRepository
	tx       pgx.Tx
	clock    *testClock
	adminID  uuid.UUID
}

func newBidFixture(t *testing.T) *bidFixture {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	clock := &testClock{now: time.Date(2026, 10, 22, 11, 0, 0, 0, time.UTC)}
	auctions := auctionrepo.NewAuctionRepositoryImpl()
	bids := bidrepo.NewBidRepositoryImpl()

	return &bidFixture{
		svc: service.NewBidService(
			auctions,
			bids,
			singleTxManager{tx: tx},
			clock.Now,
		),
		auctions: auctions,
		bids:     bids,
		tx:       tx,
		clock:    clock,
		adminID:  createAuctionTestUser(t, tx, "Admin", "admin"),
	}
}

func (f *bidFixture) createLiveAuction(t *testing.T, remaining time.Duration) domain.Auction {
	t.Helper()

	now := f.clock.Now().UTC()
	endsAt := now.Add(remaining)

	auction, err := f.auctions.Create(context.Background(), f.tx, domain.Auction{
		CreatedBy:       f.adminID,
		Lot:             "LOT-" + uuid.NewString()[:8],
		Title:           "Bid test auction",
		Description:     "Auction for bid service tests",
		Condition:       "Used",
		Includes:        []string{},
		Specs:           []domain.Spec{},
		ImageURLs:       []string{},
		StartingPrice:   1000,
		MinIncrement:    100,
		DurationSeconds: 600,
		CurrentPrice:    1000,
		Status:          domain.AuctionStatusLive,
		StartsAt:        &now,
		EndsAt:          &endsAt,
	})
	if err != nil {
		t.Fatalf("create live auction fixture: %v", err)
	}
	return auction
}

func (f *bidFixture) createBidder(t *testing.T) uuid.UUID {
	t.Helper()
	return createAuctionTestUser(t, f.tx, "Bidder", "user")
}

func (f *bidFixture) placeBid(
	t *testing.T,
	userID uuid.UUID,
	auctionID uuid.UUID,
	amount int64,
	key uuid.UUID,
) (bid_response.PlaceBidResponse, error) {
	t.Helper()

	return f.svc.PlaceBid(context.Background(), userID, auctionID, bid_request.PlaceBidRequest{
		Amount:         amount,
		IdempotencyKey: key,
	})
}

func TestBidServiceFirstBid(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 10*time.Minute)
	bidder := f.createBidder(t)

	result, err := f.placeBid(t, bidder, auction.ID, 1000, uuid.New())
	if err != nil {
		t.Fatalf("PlaceBid() error = %v", err)
	}

	if result.Replay {
		t.Error("Replay = true, want false")
	}
	if result.Extended {
		t.Error("Extended = true, want false")
	}
	if result.Auction.CurrentPrice != 1000 {
		t.Errorf("CurrentPrice = %d, want 1000", result.Auction.CurrentPrice)
	}
	if result.Auction.HighestBidderID == nil || *result.Auction.HighestBidderID != bidder {
		t.Errorf("HighestBidderID = %v, want %v", result.Auction.HighestBidderID, bidder)
	}
	if result.Auction.Version != auction.Version+1 {
		t.Errorf("Version = %d, want %d", result.Auction.Version, auction.Version+1)
	}
	if result.Bid.Amount != 1000 {
		t.Errorf("Bid.Amount = %d, want 1000", result.Bid.Amount)
	}
}

func TestBidServiceRejectsLowBid(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 10*time.Minute)
	bidder := f.createBidder(t)

	_, err := f.placeBid(t, bidder, auction.ID, 900, uuid.New())
	if !errors.Is(err, exception.ErrBidTooLow) {
		t.Fatalf("PlaceBid() error = %v, want %v", err, exception.ErrBidTooLow)
	}
}

func TestBidServiceRejectsCreator(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 10*time.Minute)

	_, err := f.placeBid(t, f.adminID, auction.ID, 1000, uuid.New())
	if !errors.Is(err, exception.ErrCreatorCannotBid) {
		t.Fatalf("PlaceBid() error = %v, want %v", err, exception.ErrCreatorCannotBid)
	}
}

func TestBidServiceRejectsAlreadyHighest(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 10*time.Minute)
	bidder := f.createBidder(t)

	if _, err := f.placeBid(t, bidder, auction.ID, 1000, uuid.New()); err != nil {
		t.Fatalf("first PlaceBid() error = %v", err)
	}

	_, err := f.placeBid(t, bidder, auction.ID, 1100, uuid.New())
	if !errors.Is(err, exception.ErrAlreadyHighest) {
		t.Fatalf("second PlaceBid() error = %v, want %v", err, exception.ErrAlreadyHighest)
	}
}

func TestBidServiceRejectsAfterEndsAt(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, time.Minute)
	bidder := f.createBidder(t)
	f.clock.Advance(2 * time.Minute)

	_, err := f.placeBid(t, bidder, auction.ID, 1000, uuid.New())
	if !errors.Is(err, exception.ErrAuctionNotLive) {
		t.Fatalf("PlaceBid() error = %v, want %v", err, exception.ErrAuctionNotLive)
	}
}

func TestBidServiceAntiSniping(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 30*time.Second)
	bidder := f.createBidder(t)
	originalEnd := *auction.EndsAt

	result, err := f.placeBid(t, bidder, auction.ID, 1000, uuid.New())
	if err != nil {
		t.Fatalf("PlaceBid() error = %v", err)
	}
	if !result.Extended {
		t.Error("Extended = false, want true")
	}
	if result.Auction.ExtensionCount != 1 {
		t.Errorf("ExtensionCount = %d, want 1", result.Auction.ExtensionCount)
	}
	if result.Auction.EndsAt == nil || !result.Auction.EndsAt.Equal(originalEnd.Add(time.Minute)) {
		t.Errorf("EndsAt = %v, want %v", result.Auction.EndsAt, originalEnd.Add(time.Minute))
	}
}

func TestBidServiceStopsAtFiveExtensions(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 30*time.Second)
	firstBidder := f.createBidder(t)
	secondBidder := f.createBidder(t)

	if _, err := f.placeBid(t, firstBidder, auction.ID, 1000, uuid.New()); err != nil {
		t.Fatalf("first PlaceBid() error = %v", err)
	}

	current, err := f.auctions.LockByID(context.Background(), f.tx, auction.ID)
	if err != nil {
		t.Fatalf("LockByID() error = %v", err)
	}
	current.ExtensionCount = 5
	if _, err := f.auctions.Update(context.Background(), f.tx, current); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	result, err := f.placeBid(t, secondBidder, auction.ID, 1100, uuid.New())
	if err != nil {
		t.Fatalf("second PlaceBid() error = %v", err)
	}
	if result.Extended {
		t.Error("Extended = true after 5 extensions")
	}
}

func TestBidServiceIdempotency(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 10*time.Minute)
	bidder := f.createBidder(t)
	key := uuid.New()

	first, err := f.placeBid(t, bidder, auction.ID, 1000, key)
	if err != nil {
		t.Fatalf("first PlaceBid() error = %v", err)
	}

	replay, err := f.placeBid(t, bidder, auction.ID, 1000, key)
	if err != nil {
		t.Fatalf("replay PlaceBid() error = %v", err)
	}
	if !replay.Replay {
		t.Error("Replay = false, want true")
	}
	if replay.Bid.ID != first.Bid.ID {
		t.Errorf("replay bid ID = %v, want %v", replay.Bid.ID, first.Bid.ID)
	}

	bids, err := f.bids.ListByAuction(context.Background(), f.tx, auction.ID, nil, 10)
	if err != nil {
		t.Fatalf("ListByAuction() error = %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("bid count = %d, want 1", len(bids))
	}

	_, err = f.placeBid(t, bidder, auction.ID, 1100, key)
	if !errors.Is(err, exception.ErrIdempotencyConflict) {
		t.Fatalf("conflict PlaceBid() error = %v, want %v", err, exception.ErrIdempotencyConflict)
	}
}

func TestBidServiceInvalidAmount(t *testing.T) {
	f := newBidFixture(t)
	auction := f.createLiveAuction(t, 10*time.Minute)
	bidder := f.createBidder(t)

	for _, amount := range []int64{0, -1, 1_000_000_000_001} {
		_, err := f.placeBid(t, bidder, auction.ID, amount, uuid.New())
		if !errors.Is(err, exception.ErrInvalidAmount) {
			t.Errorf("amount %d error = %v, want %v", amount, err, exception.ErrInvalidAmount)
		}
	}
}

func TestBidServiceAuctionNotFound(t *testing.T) {
	f := newBidFixture(t)
	bidder := f.createBidder(t)

	_, err := f.placeBid(t, bidder, uuid.New(), 1000, uuid.New())
	if !errors.Is(err, exception.ErrAuctionNotFound) {
		t.Fatalf("PlaceBid() error = %v, want %v", err, exception.ErrAuctionNotFound)
	}
}

func TestBidServiceConcurrentSameAmount(t *testing.T) {
	pool := helper.OpenTestDB(t)
	ctx := context.Background()
	txManager := repository.NewTxManager(pool)
	auctions := auctionrepo.NewAuctionRepositoryImpl()
	bids := bidrepo.NewBidRepositoryImpl()
	svc := service.NewBidService(auctions, bids, txManager, time.Now)

	var adminID, firstBidderID, secondBidderID, auctionID uuid.UUID
	if err := txManager.WithinTx(ctx, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		endsAt := now.Add(10 * time.Minute)

		admin, err := insertBidTestUser(ctx, tx, "Admin")
		if err != nil {
			return err
		}
		first, err := insertBidTestUser(ctx, tx, "First")
		if err != nil {
			return err
		}
		second, err := insertBidTestUser(ctx, tx, "Second")
		if err != nil {
			return err
		}

		auction, err := auctions.Create(ctx, tx, domain.Auction{
			CreatedBy:       admin,
			Lot:             "LOT-" + uuid.NewString()[:8],
			Title:           "Concurrent auction",
			Description:     "Concurrent test",
			Condition:       "Used",
			StartingPrice:   1000,
			MinIncrement:    100,
			DurationSeconds: 600,
			CurrentPrice:    1000,
			Status:          domain.AuctionStatusLive,
			StartsAt:        &now,
			EndsAt:          &endsAt,
		})
		if err != nil {
			return err
		}

		adminID, firstBidderID, secondBidderID, auctionID = admin, first, second, auction.ID
		return nil
	}); err != nil {
		t.Fatalf("setup transaction: %v", err)
	}

	defer func() {
		_, _ = pool.Exec(ctx, "delete from bids where auction_id = $1", auctionID)
		_, _ = pool.Exec(ctx, "delete from auctions where id = $1", auctionID)
		for _, id := range []uuid.UUID{adminID, firstBidderID, secondBidderID} {
			_, _ = pool.Exec(ctx, "delete from users where id = $1", id)
		}
	}()

	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, bidderID := range []uuid.UUID{firstBidderID, secondBidderID} {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			_, err := svc.PlaceBid(ctx, id, auctionID, bid_request.PlaceBidRequest{
				Amount:         1000,
				IdempotencyKey: uuid.New(),
			})
			results <- err
		}(bidderID)
	}
	wg.Wait()
	close(results)

	var success, tooLow int
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, exception.ErrBidTooLow):
			tooLow++
		default:
			t.Errorf("unexpected PlaceBid() error = %v", err)
		}
	}
	if success != 1 || tooLow != 1 {
		t.Fatalf("success = %d, tooLow = %d, want 1 and 1", success, tooLow)
	}
}

func insertBidTestUser(ctx context.Context, tx pgx.Tx, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(
		ctx,
		`insert into users (email, password_hash, display_name, role)
		 values ($1, 'hash', $2, 'user')
		 returning id`,
		uuid.NewString()+"@example.com",
		name,
	).Scan(&id)
	return id, err
}
