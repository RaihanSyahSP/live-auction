package controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/model/web/bid_request"
	"live-auction/server/internal/model/web/bid_response"
	"live-auction/server/internal/realtime"
)

type fakeBidService struct {
	result bid_response.PlaceBidResponse
	err    error
}

func (f *fakeBidService) PlaceBid(
	ctx context.Context,
	userID uuid.UUID,
	auctionID uuid.UUID,
	req bid_request.PlaceBidRequest,
) (bid_response.PlaceBidResponse, error) {
	return f.result, f.err
}

func withBidder(req *http.Request) *http.Request {
	return req.WithContext(middleware.WithPrincipal(req.Context(), auth.Principal{
		UserID: uuid.New(),
		Role:   domain.RoleUser,
	}))
}

func bidRequestBody(amount int64) string {
	return `{"amount":` + strconv.FormatInt(amount, 10) + `,"idempotency_key":"` + uuid.NewString() + `"}`
}

func auctionResponseForTest() auction_response.AuctionResponse {
	return auction_response.AuctionResponse{
		ID:    uuid.New(),
		Title: "Test auction",
	}
}

func bidResponseForTest() auction_response.BidResponse {
	return auction_response.BidResponse{
		ID:     uuid.New(),
		Amount: 1000,
	}
}

func TestBidControllerPlaceBid(t *testing.T) {
	svc := &fakeBidService{
		result: bid_response.PlaceBidResponse{
			Auction: auctionResponseForTest(),
			Bid:     bidResponseForTest(),
		},
	}
	handler := controller.NewBidController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/"+uuid.NewString()+"/bids",
		strings.NewReader(bidRequestBody(1000)),
	)
	req.SetPathValue("id", uuid.NewString())
	req = withBidder(req)
	rec := httptest.NewRecorder()

	handler.PlaceBid(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestBidControllerPublishesBidEvent(t *testing.T) {
	auctionID := uuid.New()
	bid := bidResponseForTest()
	bid.AuctionID = auctionID
	auction := auctionResponseForTest()
	auction.ID = auctionID
	auction.Version = 1

	svc := &fakeBidService{
		result: bid_response.PlaceBidResponse{
			Auction: auction,
			Bid:     bid,
		},
	}
	rooms := realtime.NewRoomManager()
	client, _ := rooms.Join(auctionID, nil, realtime.Snapshot{
		Auction: auction,
		Version: 0,
	})
	drainRealtimeClient(client)

	handler := controller.NewBidController(svc, rooms)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/"+auctionID.String()+"/bids",
		strings.NewReader(bidRequestBody(1000)),
	)
	req.SetPathValue("id", auctionID.String())
	req = withBidder(req)
	rec := httptest.NewRecorder()

	handler.PlaceBid(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	event := receiveRealtimeEvent(t, client)
	if event.Type != "bid" {
		t.Fatalf("event type = %q, want bid", event.Type)
	}
	if event.AuctionID != auctionID {
		t.Errorf("event auction id = %s, want %s", event.AuctionID, auctionID)
	}
	if event.Version != 1 {
		t.Errorf("event version = %d, want 1", event.Version)
	}
}

func TestBidControllerReplay(t *testing.T) {
	svc := &fakeBidService{
		result: bid_response.PlaceBidResponse{
			Auction: auctionResponseForTest(),
			Bid:     bidResponseForTest(),
			Replay:  true,
		},
	}
	handler := controller.NewBidController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/"+uuid.NewString()+"/bids",
		strings.NewReader(bidRequestBody(1000)),
	)
	req.SetPathValue("id", uuid.NewString())
	req = withBidder(req)
	rec := httptest.NewRecorder()

	handler.PlaceBid(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestBidControllerUnauthenticated(t *testing.T) {
	svc := &fakeBidService{}
	handler := controller.NewBidController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/"+uuid.NewString()+"/bids",
		strings.NewReader(bidRequestBody(1000)),
	)
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	handler.PlaceBid(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBidControllerInvalidAuctionID(t *testing.T) {
	svc := &fakeBidService{}
	handler := controller.NewBidController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/bad/bids",
		strings.NewReader(bidRequestBody(1000)),
	)
	req.SetPathValue("id", "bad")
	req = withBidder(req)
	rec := httptest.NewRecorder()

	handler.PlaceBid(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if code := errorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Errorf("error code = %q, want VALIDATION_ERROR", code)
	}
}

func TestBidControllerMalformedJSON(t *testing.T) {
	svc := &fakeBidService{}
	handler := controller.NewBidController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/"+uuid.NewString()+"/bids",
		strings.NewReader(`{`),
	)
	req.SetPathValue("id", uuid.NewString())
	req = withBidder(req)
	rec := httptest.NewRecorder()

	handler.PlaceBid(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestBidControllerErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "too low", err: exception.ErrBidTooLow, status: http.StatusUnprocessableEntity, code: "BID_TOO_LOW"},
		{name: "creator", err: exception.ErrCreatorCannotBid, status: http.StatusForbidden, code: "CREATOR_CANNOT_BID"},
		{name: "already highest", err: exception.ErrAlreadyHighest, status: http.StatusConflict, code: "ALREADY_HIGHEST"},
		{name: "not live", err: exception.ErrAuctionNotLive, status: http.StatusConflict, code: "AUCTION_NOT_LIVE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeBidService{err: tt.err}
			handler := controller.NewBidController(svc, realtime.NewRoomManager())
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/auctions/"+uuid.NewString()+"/bids",
				strings.NewReader(bidRequestBody(1000)),
			)
			req.SetPathValue("id", uuid.NewString())
			req = withBidder(req)
			rec := httptest.NewRecorder()

			handler.PlaceBid(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if code := errorCode(t, rec); code != tt.code {
				t.Errorf("error code = %q, want %q", code, tt.code)
			}
		})
	}
}
