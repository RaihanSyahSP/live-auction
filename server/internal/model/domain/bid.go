package domain

import (
	"time"

	"github.com/google/uuid"
)

type Bid struct {
	ID             uuid.UUID
	AuctionID      uuid.UUID
	UserID         uuid.UUID
	Amount         int64
	IdempotencyKey uuid.UUID
	CreatedAt      time.Time
}

type BidView struct {
	Bid
	DisplayName string
}
