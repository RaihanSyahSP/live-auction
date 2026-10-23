package realtime

import (
	"live-auction/server/internal/metrics"
	"live-auction/server/internal/model/web/auction_response"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RoomManager struct {
	mu      sync.Mutex
	rooms   map[uuid.UUID]*Room
	now     func() time.Time
	metrics *metrics.Metrics
}

func NewRoomManager(instrument ...*metrics.Metrics) *RoomManager {
	m := &RoomManager{
		rooms: make(map[uuid.UUID]*Room),
		now:   time.Now,
	}
	if len(instrument) > 0 {
		m.metrics = instrument[0]
	}
	return m
}

func (m *RoomManager) Join(
	auctionID uuid.UUID,
	userID *uuid.UUID,
	fallback Snapshot,
) (*Client, Snapshot) {

	m.mu.Lock()
	room := m.rooms[auctionID]
	if room == nil {
		room = newRoom(uuid.NewString(), fallback)
		m.rooms[auctionID] = room
	}
	m.mu.Unlock()

	room.mu.Lock()
	defer room.mu.Unlock()

	client := &Client{
		UserID: userID,
		Send:   make(chan []byte, 32),
	}
	room.clients[client] = struct{}{}
	m.metrics.SetConnectedClients(auctionID.String(), len(room.clients))

	snapshot := Snapshot{
		Type:        "snapshot",
		Epoch:       room.epoch,
		Seq:         room.seq,
		Version:     room.version,
		ServerTime:  m.now().UTC(),
		Auction:     room.auction,
		Bids:        room.bids,
		ViewerCount: len(room.clients),
	}

	m.publishPresenceLocked(room, auctionID)
	return client, snapshot
}

func (m *RoomManager) Leave(auctionID uuid.UUID, client *Client) {
	m.mu.Lock()
	room := m.rooms[auctionID]
	m.mu.Unlock()
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if _, ok := room.clients[client]; !ok {
		return
	}

	delete(room.clients, client)
	close(client.Send)
	m.metrics.SetConnectedClients(auctionID.String(), len(room.clients))

	m.publishPresenceLocked(room, auctionID)
}

func (m *RoomManager) publishPresenceLocked(room *Room, auctionID uuid.UUID) {
	now := m.now()
	if now.Sub(room.lastPresence) < time.Second {
		return
	}
	room.lastPresence = now

	event := room.nextEvent("presence", auctionID, map[string]any{
		"viewer_count": len(room.clients),
	})
	room.broadcast(event)
}

func (m *RoomManager) PublishBid(
	auction auction_response.AuctionResponse,
	bid auction_response.BidResponse,
	previousHighestUserID *uuid.UUID,
	extended bool,
) {
	m.mu.Lock()
	room := m.rooms[auction.ID]
	m.mu.Unlock()
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if auction.Version <= room.version {
		return
	}

	started := time.Now()

	room.version = auction.Version
	room.auction = auction
	room.bids = append([]auction_response.BidResponse{bid}, room.bids...)
	if len(room.bids) > 50 {
		room.bids = room.bids[:50]
	}

	bidEvent := room.nextEvent("bid", auction.ID, map[string]any{
		"auction":  auction,
		"bid":      bid,
		"extended": extended,
	})
	room.broadcast(bidEvent)

	if previousHighestUserID != nil {
		outbidEvent := room.nextEvent("outbid", auction.ID, map[string]any{
			"auction": auction,
			"bid":     bid,
		})
		for client := range room.clients {
			if client.UserID != nil && *client.UserID == *previousHighestUserID {
				room.send(client, outbidEvent)
			}
		}
	}

	if extended {
		extendedEvent := room.nextEvent("extended", auction.ID, map[string]any{
			"ends_at":         auction.EndsAt,
			"extension_count": auction.ExtensionCount,
		})
		room.broadcast(extendedEvent)
	}

	m.metrics.ObserveBroadcast(time.Since(started))
}

func (m *RoomManager) PublishEnded(auction auction_response.AuctionResponse) {
	m.mu.Lock()
	room := m.rooms[auction.ID]
	m.mu.Unlock()
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if auction.Version <= room.version {
		return
	}

	room.version = auction.Version
	room.auction = auction

	event := room.nextEvent("ended", auction.ID, map[string]any{
		"auction": auction,
	})
	room.broadcast(event)

}
