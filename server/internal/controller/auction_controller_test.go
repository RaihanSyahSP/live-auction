package controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_request"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/realtime"
)

type fakeAuctionService struct {
	currentResult auction_response.AuctionDetailResponse
	currentErr    error
	detailResult  auction_response.AuctionDetailResponse
	detailErr     error
	bidsResult    []auction_response.BidResponse
	bidsErr       error
	resultsResult []auction_response.ResultResponse
	resultsErr    error
	listResult    []auction_response.AuctionResponse
	listErr       error
	listLimit     int
	listOffset    int
	createResult  auction_response.AuctionResponse
	createErr     error
	startResult   auction_response.AuctionResponse
	startErr      error
	endResult     auction_response.AuctionResponse
	endErr        error
	deleteErr     error
	closeDue      []domain.Auction
	closeDueErr   error
}

func (f *fakeAuctionService) Current(ctx context.Context) (auction_response.AuctionDetailResponse, error) {
	return f.currentResult, f.currentErr
}

func (f *fakeAuctionService) Detail(ctx context.Context, id uuid.UUID) (auction_response.AuctionDetailResponse, error) {
	return f.detailResult, f.detailErr
}

func (f *fakeAuctionService) Bids(ctx context.Context, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]auction_response.BidResponse, error) {
	return f.bidsResult, f.bidsErr
}

func (f *fakeAuctionService) Results(ctx context.Context, limit int) ([]auction_response.ResultResponse, error) {
	return f.resultsResult, f.resultsErr
}

func (f *fakeAuctionService) List(ctx context.Context, limit int, offset int) ([]auction_response.AuctionResponse, error) {
	f.listLimit = limit
	f.listOffset = offset
	return f.listResult, f.listErr
}

func (f *fakeAuctionService) CreateDraft(ctx context.Context, adminID uuid.UUID, req auction_request.CreateAuctionRequest) (auction_response.AuctionResponse, error) {
	return f.createResult, f.createErr
}

func (f *fakeAuctionService) Start(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error) {
	return f.startResult, f.startErr
}

func (f *fakeAuctionService) End(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error) {
	return f.endResult, f.endErr
}

func (f *fakeAuctionService) DeleteDraft(ctx context.Context, adminID uuid.UUID, id uuid.UUID) error {
	return f.deleteErr
}

func (f *fakeAuctionService) CloseDue(ctx context.Context) ([]domain.Auction, error) {
	return f.closeDue, f.closeDueErr
}

func withAdmin(req *http.Request) *http.Request {
	return req.WithContext(middleware.WithPrincipal(req.Context(), auth.Principal{
		UserID: uuid.New(),
		Role:   domain.RoleAdmin,
	}))
}

func TestAuctionControllerCurrent(t *testing.T) {
	svc := &fakeAuctionService{
		currentResult: auction_response.AuctionDetailResponse{
			Auction: auction_response.AuctionResponse{ID: uuid.New(), Title: "Live"},
		},
	}
	handler := controller.NewAuctionController(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/auctions/current", nil)
	rec := httptest.NewRecorder()

	handler.Current(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuctionControllerCurrentNoLive(t *testing.T) {
	svc := &fakeAuctionService{currentErr: exception.ErrAuctionNotFound}
	handler := controller.NewAuctionController(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/auctions/current", nil)
	rec := httptest.NewRecorder()

	handler.Current(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuctionControllerDetailInvalidID(t *testing.T) {
	svc := &fakeAuctionService{}
	handler := controller.NewAuctionController(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/auctions/bad", nil)
	req.SetPathValue("id", "bad")
	rec := httptest.NewRecorder()

	handler.Detail(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if code := errorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Errorf("error code = %q, want VALIDATION_ERROR", code)
	}
}

func TestAuctionControllerBidsInvalidLimit(t *testing.T) {
	svc := &fakeAuctionService{}
	handler := controller.NewAuctionController(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/auctions/"+uuid.NewString()+"/bids?limit=0", nil)
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	handler.Bids(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestAuctionControllerBids(t *testing.T) {
	svc := &fakeAuctionService{
		bidsResult: []auction_response.BidResponse{{
			ID:          uuid.New(),
			AuctionID:   uuid.New(),
			UserID:      uuid.New(),
			Amount:      1500,
			DisplayName: "Raihan",
		}},
	}
	handler := controller.NewAuctionController(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/auctions/"+uuid.NewString()+"/bids?limit=10", nil)
	req.SetPathValue("id", uuid.NewString())
	rec := httptest.NewRecorder()

	handler.Bids(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAdminControllerListPagination(t *testing.T) {
	svc := &fakeAuctionService{
		listResult: []auction_response.AuctionResponse{{ID: uuid.New(), Title: "Auction"}},
	}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/auctions?page=2&limit=10", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if svc.listLimit != 10 {
		t.Errorf("limit = %d, want 10", svc.listLimit)
	}
	if svc.listOffset != 10 {
		t.Errorf("offset = %d, want 10", svc.listOffset)
	}
}

func TestAdminControllerListInvalidPage(t *testing.T) {
	svc := &fakeAuctionService{}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/auctions?page=0", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestAdminControllerCreate(t *testing.T) {
	svc := &fakeAuctionService{
		createResult: auction_response.AuctionResponse{ID: uuid.New(), Title: "Draft"},
	}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/admin/auctions",
		strings.NewReader(`{
			"title":"Test auction",
			"description":"A valid auction description",
			"condition":"Used",
			"starting_price":1000,
			"min_increment":100,
			"duration_minutes":10
		}`),
	)
	req = withAdmin(req)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestAdminControllerCreateUnauthenticated(t *testing.T) {
	svc := &fakeAuctionService{}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auctions", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminControllerCreateMalformedJSON(t *testing.T) {
	svc := &fakeAuctionService{}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auctions", strings.NewReader(`{`))
	req = withAdmin(req)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}

func TestAdminControllerStartEndDelete(t *testing.T) {
	svc := &fakeAuctionService{
		startResult: auction_response.AuctionResponse{ID: uuid.New(), Title: "Started"},
		endResult:   auction_response.AuctionResponse{ID: uuid.New(), Title: "Ended"},
	}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	id := uuid.NewString()

	startReq := httptest.NewRequest(http.MethodPost, "/api/admin/auctions/"+id+"/start", nil)
	startReq.SetPathValue("id", id)
	startReq = withAdmin(startReq)
	startRec := httptest.NewRecorder()
	handler.Start(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, want %d", startRec.Code, http.StatusOK)
	}

	endReq := httptest.NewRequest(http.MethodPost, "/api/admin/auctions/"+id+"/end", nil)
	endReq.SetPathValue("id", id)
	endReq = withAdmin(endReq)
	endRec := httptest.NewRecorder()
	handler.End(endRec, endReq)
	if endRec.Code != http.StatusOK {
		t.Fatalf("end status = %d, want %d", endRec.Code, http.StatusOK)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/auctions/"+id, nil)
	deleteReq.SetPathValue("id", id)
	deleteReq = withAdmin(deleteReq)
	deleteRec := httptest.NewRecorder()
	handler.Delete(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusNoContent)
	}
}

func TestAdminControllerEndPublishesEndedEvent(t *testing.T) {
	auctionID := uuid.New()
	svc := &fakeAuctionService{
		endResult: auction_response.AuctionResponse{
			ID:      auctionID,
			Status:  domain.AuctionStatusEnded,
			Version: 2,
		},
	}
	rooms := realtime.NewRoomManager()
	client, _ := rooms.Join(auctionID, nil, realtime.Snapshot{
		Auction: auction_response.AuctionResponse{
			ID:     auctionID,
			Status: domain.AuctionStatusLive,
		},
	})
	drainRealtimeClient(client)

	handler := controller.NewAdminController(svc, rooms)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/admin/auctions/"+auctionID.String()+"/end",
		nil,
	)
	req.SetPathValue("id", auctionID.String())
	req = withAdmin(req)
	rec := httptest.NewRecorder()

	handler.End(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	event := receiveRealtimeEvent(t, client)
	if event.Type != "ended" {
		t.Fatalf("event type = %q, want ended", event.Type)
	}
	if event.Version != 2 {
		t.Errorf("event version = %d, want 2", event.Version)
	}
}

func TestAdminControllerStartInvalidID(t *testing.T) {
	svc := &fakeAuctionService{}
	handler := controller.NewAdminController(svc, realtime.NewRoomManager())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auctions/bad/start", nil)
	req.SetPathValue("id", "bad")
	req = withAdmin(req)
	rec := httptest.NewRecorder()

	handler.Start(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
}
