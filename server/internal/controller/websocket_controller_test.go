package controller_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/realtime"
)

type wsTestServer struct {
	server *httptest.Server
	rooms  *realtime.RoomManager
	tokens *auth.AccessTokens
}

func newWSTestServer(t *testing.T, svc *fakeAuctionService, allowedOrigin string) *wsTestServer {
	t.Helper()

	rooms := realtime.NewRoomManager()
	tokens := auth.NewAccessTokens(
		[]byte("test-secret"),
		"test-issuer",
		"test-audience",
		time.Minute,
	)

	wsController := controller.NewWebSocketController(rooms, svc, tokens, allowedOrigin)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/auctions/{id}", wsController.ServeWS)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &wsTestServer{
		server: server,
		rooms:  rooms,
		tokens: tokens,
	}
}

func wsURL(server *httptest.Server, auctionID string) string {
	return "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/auctions/" + auctionID
}

func (s *wsTestServer) dial(t *testing.T, auctionID string, headers http.Header) *websocket.Conn {
	t.Helper()

	header := make(http.Header, len(headers)+1)
	for key, values := range headers {
		header[key] = append([]string(nil), values...)
	}
	if header.Get("Origin") == "" {
		header.Set("Origin", "http://allowed.test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(s.server, auctionID), &websocket.DialOptions{
		HTTPHeader: header,
	})
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close(websocket.StatusNormalClosure, "test done")
	})

	return conn
}

func readWSSnapshot(t *testing.T, conn *websocket.Conn) realtime.Snapshot {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	var snapshot realtime.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.Type != "snapshot" {
		t.Fatalf("message type = %q, want snapshot", snapshot.Type)
	}

	return snapshot
}

func readWSEvent(t *testing.T, conn *websocket.Conn, want string) realtime.Event {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read %s event: %v", want, err)
		}

		var event realtime.Event
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatalf("decode %s event: %v", want, err)
		}

		if event.Type == "presence" {
			continue
		}
		if event.Type != want {
			t.Fatalf("event type = %q, want %q", event.Type, want)
		}

		return event
	}
}

func TestWebSocketControllerSendsSnapshot(t *testing.T) {
	auctionID := uuid.New()
	bidID := uuid.New()
	serverTime := time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)

	svc := &fakeAuctionService{
		detailResult: auction_response.AuctionDetailResponse{
			Auction: auction_response.AuctionResponse{
				ID:      auctionID,
				Title:   "Live auction",
				Status:  domain.AuctionStatusLive,
				Version: 3,
			},
			Bids: []auction_response.BidResponse{{
				ID:        bidID,
				AuctionID: auctionID,
				Amount:    1500,
			}},
			ServerTime: serverTime,
		},
	}

	ts := newWSTestServer(t, svc, "http://allowed.test")
	conn := ts.dial(t, auctionID.String(), nil)
	snapshot := readWSSnapshot(t, conn)

	if snapshot.Epoch == "" {
		t.Error("snapshot epoch is empty")
	}
	if snapshot.Version != 3 {
		t.Errorf("snapshot version = %d, want 3", snapshot.Version)
	}
	if snapshot.Auction.ID != auctionID {
		t.Errorf("snapshot auction id = %s, want %s", snapshot.Auction.ID, auctionID)
	}
	if snapshot.ViewerCount != 1 {
		t.Errorf("snapshot viewer count = %d, want 1", snapshot.ViewerCount)
	}
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].ID != bidID {
		t.Errorf("snapshot bids = %+v, want bid %s", snapshot.Bids, bidID)
	}
}

func TestWebSocketControllerRejectsInvalidAuctionID(t *testing.T) {
	ts := newWSTestServer(t, &fakeAuctionService{}, "http://allowed.test")

	resp, err := http.Get(ts.server.URL + "/ws/auctions/not-a-uuid")
	if err != nil {
		t.Fatalf("get websocket endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

func TestWebSocketControllerReturnsNotFoundForUnknownAuction(t *testing.T) {
	ts := newWSTestServer(t, &fakeAuctionService{
		detailErr: exception.ErrAuctionNotFound,
	}, "http://allowed.test")

	resp, err := http.Get(ts.server.URL + "/ws/auctions/" + uuid.NewString())
	if err != nil {
		t.Fatalf("get websocket endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestWebSocketControllerRejectsDisallowedOrigin(t *testing.T) {
	auctionID := uuid.New()
	svc := &fakeAuctionService{
		detailResult: auction_response.AuctionDetailResponse{
			Auction: auction_response.AuctionResponse{ID: auctionID},
		},
	}
	ts := newWSTestServer(t, svc, "http://allowed.test")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, wsURL(ts.server, auctionID.String()), &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin": []string{"http://evil.test"},
		},
	})
	if err == nil {
		_ = conn.Close(websocket.StatusNormalClosure, "unexpected connection")
		t.Fatal("websocket connection with disallowed origin was accepted")
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

func TestWebSocketControllerBroadcastsBidOutbidAndExtension(t *testing.T) {
	auctionID := uuid.New()
	previousHighestID := uuid.New()

	svc := &fakeAuctionService{
		detailResult: auction_response.AuctionDetailResponse{
			Auction: auction_response.AuctionResponse{
				ID:      auctionID,
				Status:  domain.AuctionStatusLive,
				Version: 1,
			},
		},
	}
	ts := newWSTestServer(t, svc, "http://allowed.test")

	token, err := ts.tokens.Issue(previousHighestID, domain.RoleUser)
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	previousHighest := ts.dial(t, auctionID.String(), http.Header{
		"Cookie": []string{auth.AccessCookieName + "=" + token},
	})
	other := ts.dial(t, auctionID.String(), nil)
	readWSSnapshot(t, previousHighest)
	readWSSnapshot(t, other)

	auction := auction_response.AuctionResponse{
		ID:             auctionID,
		Status:         domain.AuctionStatusLive,
		CurrentPrice:   2000,
		ExtensionCount: 1,
		Version:        2,
	}
	bid := auction_response.BidResponse{
		ID:        uuid.New(),
		AuctionID: auctionID,
		UserID:    uuid.New(),
		Amount:    2000,
	}

	ts.rooms.PublishBid(auction, bid, &previousHighestID, true)

	if event := readWSEvent(t, previousHighest, "bid"); event.Version != 2 {
		t.Errorf("previous highest bid version = %d, want 2", event.Version)
	}
	if event := readWSEvent(t, other, "bid"); event.Version != 2 {
		t.Errorf("other bid version = %d, want 2", event.Version)
	}
	if event := readWSEvent(t, previousHighest, "outbid"); event.AuctionID != auctionID {
		t.Errorf("outbid auction id = %s, want %s", event.AuctionID, auctionID)
	}
	if event := readWSEvent(t, previousHighest, "extended"); event.Version != 2 {
		t.Errorf("previous highest extended version = %d, want 2", event.Version)
	}
	if event := readWSEvent(t, other, "extended"); event.Version != 2 {
		t.Errorf("other extended version = %d, want 2", event.Version)
	}
}

func TestWebSocketControllerPublishesEnded(t *testing.T) {
	auctionID := uuid.New()
	svc := &fakeAuctionService{
		detailResult: auction_response.AuctionDetailResponse{
			Auction: auction_response.AuctionResponse{
				ID:      auctionID,
				Status:  domain.AuctionStatusLive,
				Version: 1,
			},
		},
	}
	ts := newWSTestServer(t, svc, "http://allowed.test")
	conn := ts.dial(t, auctionID.String(), nil)
	readWSSnapshot(t, conn)

	endedAt := time.Now().UTC()
	ts.rooms.PublishEnded(auction_response.AuctionResponse{
		ID:      auctionID,
		Status:  domain.AuctionStatusEnded,
		Version: 2,
		EndedAt: &endedAt,
	})

	event := readWSEvent(t, conn, "ended")
	if event.Version != 2 {
		t.Errorf("event version = %d, want 2", event.Version)
	}

	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want map[string]any", event.Payload)
	}
	auctionPayload, ok := payload["auction"].(map[string]any)
	if !ok {
		t.Fatalf("auction payload type = %T, want map[string]any", payload["auction"])
	}
	if auctionPayload["status"] != string(domain.AuctionStatusEnded) {
		t.Errorf("auction status = %v, want ended", auctionPayload["status"])
	}
}

func TestWebSocketControllerReconnectUsesCurrentSnapshot(t *testing.T) {
	auctionID := uuid.New()
	svc := &fakeAuctionService{
		detailResult: auction_response.AuctionDetailResponse{
			Auction: auction_response.AuctionResponse{
				ID:      auctionID,
				Status:  domain.AuctionStatusLive,
				Version: 1,
			},
		},
	}
	ts := newWSTestServer(t, svc, "http://allowed.test")

	first := ts.dial(t, auctionID.String(), nil)
	firstSnapshot := readWSSnapshot(t, first)
	if firstSnapshot.Version != 1 {
		t.Fatalf("first snapshot version = %d, want 1", firstSnapshot.Version)
	}

	bid := auction_response.BidResponse{
		ID:        uuid.New(),
		AuctionID: auctionID,
		UserID:    uuid.New(),
		Amount:    1500,
	}
	ts.rooms.PublishBid(auction_response.AuctionResponse{
		ID:           auctionID,
		Status:       domain.AuctionStatusLive,
		CurrentPrice: 1500,
		Version:      2,
	}, bid, nil, false)
	readWSEvent(t, first, "bid")

	if err := first.Close(websocket.StatusNormalClosure, "reconnect"); err != nil {
		t.Fatalf("close first connection: %v", err)
	}

	second := ts.dial(t, auctionID.String(), nil)
	secondSnapshot := readWSSnapshot(t, second)

	if secondSnapshot.Version != 2 {
		t.Errorf("reconnected snapshot version = %d, want 2", secondSnapshot.Version)
	}
	if len(secondSnapshot.Bids) != 1 || secondSnapshot.Bids[0].ID != bid.ID {
		t.Errorf("reconnected snapshot bids = %+v, want bid %s", secondSnapshot.Bids, bid.ID)
	}
}
