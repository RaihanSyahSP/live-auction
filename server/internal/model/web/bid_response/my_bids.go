package bid_response

import (
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/model/domain"
)

type MyBidAuctionResponse struct {
	ID           uuid.UUID            `json:"id"`
	Lot          string               `json:"lot"`
	Title        string               `json:"title"`
	Subtitle     string               `json:"subtitle"`
	Status       domain.AuctionStatus `json:"status"`
	CurrentPrice int64                `json:"current_price"`
	FinalPrice   *int64               `json:"final_price,omitempty"`
	EndsAt       *time.Time           `json:"ends_at,omitempty"`
	EndedAt      *time.Time           `json:"ended_at,omitempty"`
}

type MyBidResponse struct {
	ID        uuid.UUID            `json:"id"`
	Amount    int64                `json:"amount"`
	CreatedAt time.Time            `json:"created_at"`
	Status    domain.BidStatus     `json:"status"`
	Auction   MyBidAuctionResponse `json:"auction"`
}

type MyBidsPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type MyBidsResponse struct {
	Bids       []MyBidResponse  `json:"bids"`
	Pagination MyBidsPagination `json:"pagination"`
}
