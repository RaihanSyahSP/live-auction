package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_response"
)

func testTime() time.Time {
	return time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)
}

func testAuction(id uuid.UUID, version int64) auction_response.AuctionResponse {
	endsAt := testTime().Add(10 * time.Minute)
	return auction_response.AuctionResponse{
		ID:           id,
		Title:        "Test auction",
		Status:       domain.AuctionStatusLive,
		CurrentPrice: 1000,
		EndsAt:       &endsAt,
		Version:      version,
	}
}

func testBid(auctionID, userID uuid.UUID, amount int64) auction_response.BidResponse {
	return auction_response.BidResponse{
		ID:        uuid.New(),
		AuctionID: auctionID,
		UserID:    userID,
		Amount:    amount,
		CreatedAt: testTime(),
	}
}

func newTestManager(now *time.Time) *RoomManager {
	m := NewRoomManager()
	m.now = func() time.Time { return *now }
	return m
}

func decodeEvent(t *testing.T, data []byte) Event {
	t.Helper()

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	return event
}

func receiveEvent(t *testing.T, client *Client) Event {
	t.Helper()

	select {
	case data, ok := <-client.Send:
		if !ok {
			t.Fatal("client send channel closed")
		}
		return decodeEvent(t, data)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func drainClient(client *Client) {
	for {
		select {
		case <-client.Send:
		default:
			return
		}
	}
}

func eventPayload(t *testing.T, event Event) map[string]any {
	t.Helper()

	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want map[string]any", event.Payload)
	}
	return payload
}

func TestRoomManagerJoinReturnsSnapshot(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()
	bid := testBid(auctionID, uuid.New(), 1200)

	client, snapshot := m.Join(auctionID, nil, Snapshot{
		Type:       "snapshot",
		Seq:        4,
		Version:    3,
		ServerTime: now,
		Auction:    testAuction(auctionID, 3),
		Bids:       []auction_response.BidResponse{bid},
	})

	if snapshot.Type != "snapshot" {
		t.Errorf("snapshot type = %q, want snapshot", snapshot.Type)
	}
	if snapshot.Epoch == "" {
		t.Error("snapshot epoch is empty")
	}
	if snapshot.Seq != 4 {
		t.Errorf("snapshot seq = %d, want 4", snapshot.Seq)
	}
	if snapshot.Version != 3 {
		t.Errorf("snapshot version = %d, want 3", snapshot.Version)
	}
	if !snapshot.ServerTime.Equal(now) {
		t.Errorf("server time = %s, want %s", snapshot.ServerTime, now)
	}
	if snapshot.Auction.ID != auctionID {
		t.Errorf("auction id = %s, want %s", snapshot.Auction.ID, auctionID)
	}
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].ID != bid.ID {
		t.Errorf("snapshot bids = %+v, want one bid %s", snapshot.Bids, bid.ID)
	}
	if snapshot.ViewerCount != 1 {
		t.Errorf("viewer count = %d, want 1", snapshot.ViewerCount)
	}
	if client.Send == nil {
		t.Error("client send channel is nil")
	}
}

func TestRoomManagerJoinKeepsExistingRoomState(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	first, firstSnapshot := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(first)

	_, secondSnapshot := m.Join(auctionID, nil, Snapshot{
		Version: 99,
		Auction: testAuction(auctionID, 99),
	})

	if secondSnapshot.Epoch != firstSnapshot.Epoch {
		t.Errorf("epoch changed from %q to %q", firstSnapshot.Epoch, secondSnapshot.Epoch)
	}
	if secondSnapshot.Version != 1 {
		t.Errorf("version = %d, want room version 1", secondSnapshot.Version)
	}
	if secondSnapshot.ViewerCount != 2 {
		t.Errorf("viewer count = %d, want 2", secondSnapshot.ViewerCount)
	}
}

func TestRoomManagerPublishBidBroadcasts(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	first, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	second, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(first)
	drainClient(second)

	auction := testAuction(auctionID, 2)
	auction.CurrentPrice = 1500
	bid := testBid(auctionID, uuid.New(), 1500)

	m.PublishBid(auction, bid, nil, false)

	for name, client := range map[string]*Client{"first": first, "second": second} {
		event := receiveEvent(t, client)
		if event.Type != "bid" {
			t.Fatalf("%s event type = %q, want bid", name, event.Type)
		}
		if event.Version != 2 {
			t.Errorf("%s event version = %d, want 2", name, event.Version)
		}
		if event.AuctionID != auctionID {
			t.Errorf("%s event auction id = %s, want %s", name, event.AuctionID, auctionID)
		}

		payload := eventPayload(t, event)
		bidPayload, ok := payload["bid"].(map[string]any)
		if !ok {
			t.Fatalf("%s bid payload = %T, want map", name, payload["bid"])
		}
		if bidPayload["amount"] != float64(1500) {
			t.Errorf("%s bid amount = %v, want 1500", name, bidPayload["amount"])
		}
	}

	_, snapshot := m.Join(auctionID, nil, Snapshot{})
	if snapshot.Version != 2 {
		t.Errorf("snapshot version = %d, want 2", snapshot.Version)
	}
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].Amount != 1500 {
		t.Errorf("snapshot bids = %+v, want one bid of 1500", snapshot.Bids)
	}
}

func TestRoomManagerOutbidOnlyGoesToPreviousHighest(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()
	previousHighestID := uuid.New()

	previousHighest, _ := m.Join(auctionID, &previousHighestID, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	other, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(previousHighest)
	drainClient(other)

	auction := testAuction(auctionID, 2)
	bid := testBid(auctionID, uuid.New(), 1500)
	m.PublishBid(auction, bid, &previousHighestID, false)

	if event := receiveEvent(t, previousHighest); event.Type != "bid" {
		t.Fatalf("previous highest event = %q, want bid", event.Type)
	}
	if event := receiveEvent(t, other); event.Type != "bid" {
		t.Fatalf("other event = %q, want bid", event.Type)
	}
	if event := receiveEvent(t, previousHighest); event.Type != "outbid" {
		t.Fatalf("previous highest event = %q, want outbid", event.Type)
	}
	if len(other.Send) != 0 {
		t.Fatalf("other client received %d unexpected event(s)", len(other.Send))
	}
}

func TestRoomManagerPublishBidBroadcastsExtension(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	first, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	second, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(first)
	drainClient(second)

	auction := testAuction(auctionID, 2)
	auction.ExtensionCount = 1
	endsAt := testTime().Add(15 * time.Minute)
	auction.EndsAt = &endsAt
	bid := testBid(auctionID, uuid.New(), 1500)

	m.PublishBid(auction, bid, nil, true)

	for name, client := range map[string]*Client{"first": first, "second": second} {
		if event := receiveEvent(t, client); event.Type != "bid" {
			t.Fatalf("%s event = %q, want bid", name, event.Type)
		}

		event := receiveEvent(t, client)
		if event.Type != "extended" {
			t.Fatalf("%s event = %q, want extended", name, event.Type)
		}
		payload := eventPayload(t, event)
		if payload["extension_count"] != float64(1) {
			t.Errorf("%s extension count = %v, want 1", name, payload["extension_count"])
		}
		if payload["ends_at"] == nil {
			t.Errorf("%s ends_at is nil", name)
		}
	}
}

func TestRoomManagerIgnoresStaleBid(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	client, _ := m.Join(auctionID, nil, Snapshot{
		Version: 3,
		Auction: testAuction(auctionID, 3),
	})
	drainClient(client)

	m.PublishBid(
		testAuction(auctionID, 3),
		testBid(auctionID, uuid.New(), 1500),
		nil,
		false,
	)

	if len(client.Send) != 0 {
		t.Fatalf("client received %d event(s) for stale version", len(client.Send))
	}

	_, snapshot := m.Join(auctionID, nil, Snapshot{})
	if snapshot.Version != 3 {
		t.Errorf("snapshot version = %d, want 3", snapshot.Version)
	}
	if len(snapshot.Bids) != 0 {
		t.Errorf("snapshot bids = %+v, want none", snapshot.Bids)
	}
}

func TestRoomManagerPublishEndedBroadcasts(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	first, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	second, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(first)
	drainClient(second)

	ended := testAuction(auctionID, 2)
	ended.Status = domain.AuctionStatusEnded
	endedAt := testTime()
	ended.EndedAt = &endedAt

	m.PublishEnded(ended)

	for name, client := range map[string]*Client{"first": first, "second": second} {
		event := receiveEvent(t, client)
		if event.Type != "ended" {
			t.Fatalf("%s event = %q, want ended", name, event.Type)
		}
		if event.Version != 2 {
			t.Errorf("%s event version = %d, want 2", name, event.Version)
		}
		payload := eventPayload(t, event)
		auctionPayload, ok := payload["auction"].(map[string]any)
		if !ok {
			t.Fatalf("%s auction payload = %T, want map", name, payload["auction"])
		}
		if auctionPayload["status"] != string(domain.AuctionStatusEnded) {
			t.Errorf("%s auction status = %v, want ended", name, auctionPayload["status"])
		}
	}

	_, snapshot := m.Join(auctionID, nil, Snapshot{})
	if snapshot.Version != 2 {
		t.Errorf("snapshot version = %d, want 2", snapshot.Version)
	}
	if snapshot.Auction.Status != domain.AuctionStatusEnded {
		t.Errorf("snapshot status = %q, want ended", snapshot.Auction.Status)
	}
}

func TestRoomManagerPublishEndedIgnoresDuplicateVersion(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	client, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(client)

	ended := testAuction(auctionID, 2)
	ended.Status = domain.AuctionStatusEnded

	m.PublishEnded(ended)
	if event := receiveEvent(t, client); event.Type != "ended" {
		t.Fatalf("event = %q, want ended", event.Type)
	}

	m.PublishEnded(ended)

	if len(client.Send) != 0 {
		t.Fatal("duplicate ended event was published")
	}
}

func TestRoomManagerLeaveClosesClientAndPublishesPresence(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	first, firstSnapshot := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	if firstSnapshot.ViewerCount != 1 {
		t.Errorf("first viewer count = %d, want 1", firstSnapshot.ViewerCount)
	}

	presence := receiveEvent(t, first)
	if presence.Type != "presence" {
		t.Fatalf("event = %q, want presence", presence.Type)
	}
	if payload := eventPayload(t, presence); payload["viewer_count"] != float64(1) {
		t.Errorf("viewer count = %v, want 1", payload["viewer_count"])
	}

	now = now.Add(500 * time.Millisecond)
	second, secondSnapshot := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	if secondSnapshot.ViewerCount != 2 {
		t.Errorf("second viewer count = %d, want 2", secondSnapshot.ViewerCount)
	}
	if len(first.Send) != 0 || len(second.Send) != 0 {
		t.Fatal("presence event was not debounced")
	}

	now = now.Add(time.Second)
	m.Leave(auctionID, first)

	if _, ok := <-first.Send; ok {
		t.Fatal("first client send channel is still open")
	}

	presence = receiveEvent(t, second)
	if presence.Type != "presence" {
		t.Fatalf("event = %q, want presence", presence.Type)
	}
	if payload := eventPayload(t, presence); payload["viewer_count"] != float64(1) {
		t.Errorf("viewer count = %v, want 1", payload["viewer_count"])
	}
}
