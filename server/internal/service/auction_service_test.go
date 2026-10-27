package service_test

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
	"live-auction/server/internal/model/web/auction_request"
	"live-auction/server/internal/model/web/auction_response"
	auctionrepo "live-auction/server/internal/repository/auction"
	bidrepo "live-auction/server/internal/repository/bid"
	"live-auction/server/internal/service"
)

type auctionFixture struct {
	svc      service.AuctionService
	auctions auctionrepo.AuctionRepository
	bids     bidrepo.BidRepository
	tx       pgx.Tx
	clock    *testClock
	adminID  uuid.UUID
}

func newAuctionFixture(t *testing.T) *auctionFixture {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	clock := &testClock{now: time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)}
	auctions := auctionrepo.NewAuctionRepositoryImpl()
	bids := bidrepo.NewBidRepositoryImpl()

	return &auctionFixture{
		svc: service.NewAuctionService(
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

func createAuctionTestUser(t *testing.T, tx pgx.Tx, name, role string) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(
		context.Background(),
		`insert into users (email, password_hash, display_name, role)
		 values ($1, 'hash', $2, $3)
		 returning id`,
		uuid.NewString()+"@example.com",
		name,
		role,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	return id
}

func (f *auctionFixture) createDraft(t *testing.T) auction_response.AuctionResponse {
	t.Helper()

	result, err := f.svc.CreateDraft(context.Background(), f.adminID, auction_request.CreateAuctionRequest{
		Title:           "Test auction",
		Description:     "A valid auction description",
		Condition:       "Used",
		Includes:        []string{"Body"},
		Specs:           []auction_request.SpecRequest{{Label: "Sensor", Value: "24 MP"}},
		ImageURLs:       []string{},
		StartingPrice:   1000,
		MinIncrement:    100,
		DurationMinutes: 1,
	})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	return result
}

func (f *auctionFixture) start(t *testing.T, id uuid.UUID) auction_response.AuctionResponse {
	t.Helper()

	result, err := f.svc.Start(context.Background(), f.adminID, id)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return result
}

func (f *auctionFixture) setHighestBid(t *testing.T, auctionID uuid.UUID, amount int64) uuid.UUID {
	t.Helper()

	bidderID := createAuctionTestUser(t, f.tx, "Bidder", "user")
	ctx := context.Background()

	if _, err := f.bids.Insert(ctx, f.tx, domain.Bid{
		AuctionID:      auctionID,
		UserID:         bidderID,
		Amount:         amount,
		IdempotencyKey: uuid.New(),
	}); err != nil {
		t.Fatalf("Insert bid fixture: %v", err)
	}

	auction, err := f.auctions.LockByID(ctx, f.tx, auctionID)
	if err != nil {
		t.Fatalf("LockByID() error = %v", err)
	}
	auction.CurrentPrice = amount
	auction.HighestBidderID = &bidderID
	auction.Version++
	if _, err := f.auctions.Update(ctx, f.tx, auction); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	return bidderID
}

func TestAuctionServiceCreateDraft(t *testing.T) {
	f := newAuctionFixture(t)

	result := f.createDraft(t)

	if result.Status != domain.AuctionStatusDraft {
		t.Errorf("Status = %q, want draft", result.Status)
	}
	if result.CurrentPrice != 1000 {
		t.Errorf("CurrentPrice = %d, want 1000", result.CurrentPrice)
	}
	if result.DurationSeconds != 60 {
		t.Errorf("DurationSeconds = %d, want 60", result.DurationSeconds)
	}
	if result.Lot == "" {
		t.Error("Lot is empty")
	}
	if len(result.Specs) != 1 || result.Specs[0].Label != "Sensor" {
		t.Errorf("Specs = %+v", result.Specs)
	}
}

func TestAuctionServiceStartAndCurrent(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)

	started := f.start(t, draft.ID)

	if started.Status != domain.AuctionStatusLive {
		t.Errorf("Status = %q, want live", started.Status)
	}
	if started.StartsAt == nil || started.EndsAt == nil {
		t.Fatal("StartsAt or EndsAt is nil")
	}
	if !started.EndsAt.After(*started.StartsAt) {
		t.Error("EndsAt is not after StartsAt")
	}
	if started.Version != draft.Version+1 {
		t.Errorf("Version = %d, want %d", started.Version, draft.Version+1)
	}

	current, err := f.svc.Current(context.Background())
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if current.Auction.ID != started.ID {
		t.Errorf("Current().Auction.ID = %v, want %v", current.Auction.ID, started.ID)
	}

	detail, err := f.svc.Detail(context.Background(), started.ID)
	if err != nil {
		t.Fatalf("Detail() error = %v", err)
	}
	if detail.Auction.ID != started.ID {
		t.Errorf("Detail().Auction.ID = %v, want %v", detail.Auction.ID, started.ID)
	}
}

func TestAuctionServiceStartRejectsSecondLive(t *testing.T) {
	f := newAuctionFixture(t)
	first := f.createDraft(t)
	second := f.createDraft(t)

	f.start(t, first.ID)

	_, err := f.svc.Start(context.Background(), f.adminID, second.ID)
	if !errors.Is(err, exception.ErrAuctionAlreadyLive) {
		t.Fatalf("Start() error = %v, want %v", err, exception.ErrAuctionAlreadyLive)
	}
}

func TestAuctionServiceEndWithBid(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	started := f.start(t, draft.ID)
	bidderID := f.setHighestBid(t, started.ID, 1500)

	ended, err := f.svc.End(context.Background(), f.adminID, started.ID)
	if err != nil {
		t.Fatalf("End() error = %v", err)
	}
	if ended.Status != domain.AuctionStatusEnded {
		t.Errorf("Status = %q, want ended", ended.Status)
	}
	if ended.WinnerID == nil || *ended.WinnerID != bidderID {
		t.Errorf("WinnerID = %v, want %v", ended.WinnerID, bidderID)
	}
	if ended.FinalPrice == nil || *ended.FinalPrice != 1500 {
		t.Errorf("FinalPrice = %v, want 1500", ended.FinalPrice)
	}

	again, err := f.svc.End(context.Background(), f.adminID, started.ID)
	if err != nil {
		t.Fatalf("second End() error = %v", err)
	}
	if again.FinalPrice == nil || *again.FinalPrice != 1500 {
		t.Errorf("second FinalPrice = %v, want 1500", again.FinalPrice)
	}
}

func TestAuctionServiceEndWithoutBids(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	started := f.start(t, draft.ID)

	ended, err := f.svc.End(context.Background(), f.adminID, started.ID)
	if err != nil {
		t.Fatalf("End() error = %v", err)
	}
	if ended.WinnerID != nil {
		t.Errorf("WinnerID = %v, want nil", ended.WinnerID)
	}
	if ended.FinalPrice != nil {
		t.Errorf("FinalPrice = %v, want nil", ended.FinalPrice)
	}
}

func TestAuctionServiceDeleteDraft(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)

	if err := f.svc.DeleteDraft(context.Background(), f.adminID, draft.ID); err != nil {
		t.Fatalf("DeleteDraft() error = %v", err)
	}

	if _, err := f.svc.Detail(context.Background(), draft.ID); !errors.Is(err, exception.ErrAuctionNotFound) {
		t.Fatalf("Detail() error = %v, want %v", err, exception.ErrAuctionNotFound)
	}

	second := f.createDraft(t)
	f.start(t, second.ID)
	if err := f.svc.DeleteDraft(context.Background(), f.adminID, second.ID); !errors.Is(err, exception.ErrAuctionNotDraft) {
		t.Fatalf("DeleteDraft() error = %v, want %v", err, exception.ErrAuctionNotDraft)
	}
}

func TestAuctionServiceCloseDue(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)
	f.clock.Advance(2 * time.Minute)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 1 {
		t.Fatalf("CloseDue() length = %d, want 1", len(ended))
	}

	if _, err := f.svc.Current(context.Background()); !errors.Is(err, exception.ErrAuctionNotFound) {
		t.Fatalf("Current() error = %v, want %v", err, exception.ErrAuctionNotFound)
	}
}

func TestAuctionServiceCloseDueSkipsNotDue(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 0 {
		t.Fatalf("CloseDue() length = %d, want 0", len(ended))
	}

	if _, err := f.svc.Current(context.Background()); err != nil {
		t.Fatalf("Current() error = %v", err)
	}
}

func TestAuctionServiceCloseDueReturnsEndedResponse(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)
	f.clock.Advance(2 * time.Minute)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 1 {
		t.Fatalf("CloseDue() length = %d, want 1", len(ended))
	}
	if ended[0].Status != domain.AuctionStatusEnded {
		t.Errorf("Status = %q, want ended", ended[0].Status)
	}
	if ended[0].Version == 0 {
		t.Error("Version = 0, want incremented version")
	}
}

func TestAuctionServiceCloseDueTwice(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)
	f.clock.Advance(2 * time.Minute)

	first, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("first CloseDue() error = %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("first length = %d, want 1", len(first))
	}

	second, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("second CloseDue() error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second length = %d, want 0", len(second))
	}
}

func TestAuctionServiceCloseDueAfterAdminEnd(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	started := f.start(t, draft.ID)
	bidderID := f.setHighestBid(t, started.ID, 1500)
	f.clock.Advance(2 * time.Minute)

	adminEnded, err := f.svc.End(context.Background(), f.adminID, started.ID)
	if err != nil {
		t.Fatalf("End() error = %v", err)
	}

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 0 {
		t.Fatalf("CloseDue() length = %d, want 0", len(ended))
	}

	if adminEnded.WinnerID == nil || *adminEnded.WinnerID != bidderID {
		t.Errorf("WinnerID = %v, want %v", adminEnded.WinnerID, bidderID)
	}
}

func TestAuctionServiceCloseDueRecoversOverdueAuction(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)
	f.clock.Advance(2 * time.Minute)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 1 {
		t.Fatalf("CloseDue() length = %d, want 1", len(ended))
	}
}

func TestAuctionServiceListPagination(t *testing.T) {
	f := newAuctionFixture(t)
	f.createDraft(t)
	f.createDraft(t)
	f.createDraft(t)

	first, err := f.svc.List(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("first page length = %d, want 2", len(first))
	}

	second, err := f.svc.List(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("List() second error = %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("second page length = %d, want 1", len(second))
	}
}

func TestAuctionServiceBidsAndResults(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	started := f.start(t, draft.ID)
	f.setHighestBid(t, started.ID, 1500)

	bids, err := f.svc.Bids(context.Background(), started.ID, nil, 10)
	if err != nil {
		t.Fatalf("Bids() error = %v", err)
	}
	if len(bids) != 1 || bids[0].Amount != 1500 {
		t.Fatalf("Bids() = %+v", bids)
	}

	if _, err := f.svc.End(context.Background(), f.adminID, started.ID); err != nil {
		t.Fatalf("End() error = %v", err)
	}

	results, err := f.svc.Results(context.Background(), 10)
	if err != nil {
		t.Fatalf("Results() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Results() length = %d, want 1", len(results))
	}
	if results[0].Auction.FinalPrice == nil || *results[0].Auction.FinalPrice != 1500 {
		t.Errorf("FinalPrice = %v, want 1500", results[0].Auction.FinalPrice)
	}
	if results[0].WinnerName != "Bidder" {
		t.Errorf("WinnerName = %q, want Bidder", results[0].WinnerName)
	}
	if results[0].BidCount != 1 {
		t.Errorf("BidCount = %d, want 1", results[0].BidCount)
	}
}
