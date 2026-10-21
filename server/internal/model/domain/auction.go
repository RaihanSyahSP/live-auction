package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuctionStatus string

const (
	AuctionStatusDraft AuctionStatus = "draft"
	AuctionStatusLive  AuctionStatus = "live"
	AuctionStatusEnded AuctionStatus = "ended"
)

type Spec struct {
	Label string
	Value string
}

type Auction struct {
	ID              uuid.UUID
	CreatedBy       uuid.UUID
	Lot             string
	Title           string
	Subtitle        string
	Description     string
	Condition       string
	Includes        []string
	Specs           []Spec
	ImageURLs       []string
	StartingPrice   int64
	MinIncrement    int64
	DurationSeconds int
	CurrentPrice    int64
	HighestBidderID *uuid.UUID
	Status          AuctionStatus
	StartsAt        *time.Time
	EndsAt          *time.Time
	EndedAt         *time.Time
	ExtensionCount  int
	Version         int64
	WinnerID        *uuid.UUID
	FinalPrice      *int64
	CreatedAt       time.Time
}
