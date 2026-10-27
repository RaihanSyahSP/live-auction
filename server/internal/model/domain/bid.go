package domain

import (
	"time"

	"github.com/google/uuid"
)

type BidStatus string

const (
	BidStatusLeading BidStatus = "leading"
	BidStatusOutbid  BidStatus = "outbid"
	BidStatusWon     BidStatus = "won"
	BidStatusLost    BidStatus = "lost"
)

type Bid struct {
	ID             uuid.UUID
	AuctionID      uuid.UUID
	UserID         uuid.UUID
	Amount         int64
	IdempotencyKey uuid.UUID
	CreatedAt      time.Time
	DisplayName    string
}

type BidView struct {
	Bid
}

type MyBidAuction struct {
	ID           uuid.UUID
	Lot          string
	Title        string
	Subtitle     string
	Status       AuctionStatus
	CurrentPrice int64
	FinalPrice   *int64
	EndsAt       *time.Time
	EndedAt      *time.Time
}

type MyBid struct {
	Bid
	Status  BidStatus
	Auction MyBidAuction
}
