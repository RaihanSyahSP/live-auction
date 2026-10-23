package realtime

import (
	"encoding/json"
	"live-auction/server/internal/model/web/auction_response"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	mu           sync.Mutex
	clients      map[*Client]struct{}
	epoch        string
	seq          int64
	version      int64
	auction      auction_response.AuctionResponse
	bids         []auction_response.BidResponse
	lastPresence time.Time
}

func newRoom(epoch string, fallback Snapshot) *Room {
	return &Room{
		clients: make(map[*Client]struct{}),
		epoch:   epoch,
		seq:     fallback.Seq,
		version: fallback.Version,
		auction: fallback.Auction,
		bids:    fallback.Bids,
	}
}

func (r *Room) nextEvent(eventType string, auctionID uuid.UUID, payload any) []byte {
	r.seq++

	event := Event{
		Type:       eventType,
		AuctionID:  auctionID,
		Epoch:      r.epoch,
		Seq:        r.seq,
		Version:    r.version,
		ServerTime: time.Now().UTC(),
		Payload:    payload,
	}

	data, _ := json.Marshal(event)
	return data
}

func (r *Room) broadcast(data []byte) {
	for client := range r.clients {
		r.send(client, data)
	}
}

func (r *Room) send(client *Client, data []byte) {
	select {
	case client.Send <- data:
	default:
		delete(r.clients, client)
		close(client.Send)
	}
}
