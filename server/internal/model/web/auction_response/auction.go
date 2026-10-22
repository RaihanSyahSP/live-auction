package auction_response

import (
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/model/domain"
)

type SpecResponse struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type AuctionResponse struct {
	ID              uuid.UUID            `json:"id"`
	CreatedBy       uuid.UUID            `json:"created_by"`
	Lot             string               `json:"lot"`
	Title           string               `json:"title"`
	Subtitle        string               `json:"subtitle"`
	Description     string               `json:"description"`
	Condition       string               `json:"condition"`
	Includes        []string             `json:"includes"`
	Specs           []SpecResponse       `json:"specs"`
	ImageURLs       []string             `json:"image_urls"`
	StartingPrice   int64                `json:"starting_price"`
	MinIncrement    int64                `json:"min_increment"`
	DurationSeconds int                  `json:"duration_seconds"`
	CurrentPrice    int64                `json:"current_price"`
	HighestBidderID *uuid.UUID           `json:"highest_bidder_id,omitempty"`
	Status          domain.AuctionStatus `json:"status"`
	StartsAt        *time.Time           `json:"starts_at,omitempty"`
	EndsAt          *time.Time           `json:"ends_at,omitempty"`
	EndedAt         *time.Time           `json:"ended_at,omitempty"`
	ExtensionCount  int                  `json:"extension_count"`
	Version         int64                `json:"version"`
	WinnerID        *uuid.UUID           `json:"winner_id,omitempty"`
	FinalPrice      *int64               `json:"final_price,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
}

type BidResponse struct {
	ID          uuid.UUID `json:"id"`
	AuctionID   uuid.UUID `json:"auction_id"`
	UserID      uuid.UUID `json:"user_id"`
	Amount      int64     `json:"amount"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuctionDetailResponse struct {
	Auction    AuctionResponse `json:"auction"`
	Bids       []BidResponse   `json:"bids"`
	ServerTime time.Time       `json:"server_time"`
}

type ResultResponse struct {
	Auction AuctionResponse `json:"auction"`
}
