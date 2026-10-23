package realtime

import (
	"live-auction/server/internal/model/web/auction_response"
	"time"

	"github.com/google/uuid"
)

type Snapshot struct {
	Type        string                           `json:"type"`
	Epoch       string                           `json:"epoch"`
	Seq         int64                            `json:"seq"`
	Version     int64                            `json:"version"`
	ServerTime  time.Time                        `json:"server_time"`
	Auction     auction_response.AuctionResponse `json:"auction"`
	Bids        []auction_response.BidResponse   `json:"bids"`
	ViewerCount int                              `json:"viewer_count"`
}

type Event struct {
	Type       string    `json:"type"`
	AuctionID  uuid.UUID `json:"auction_id"`
	Epoch      string    `json:"epoch"`
	Seq        int64     `json:"seq"`
	Version    int64     `json:"version"`
	ServerTime time.Time `json:"server_time"`
	Payload    any       `json:"payload"`
}
