package auction

import (
	"live-auction/server/internal/model/domain"
	"time"
)

func MinimumBid(a domain.Auction) int64 {
	if a.HighestBidderID == nil {
		return a.StartingPrice
	}
	return a.CurrentPrice + a.MinIncrement
}

func NextEndsAt(now, endsAt time.Time, extensions int) (time.Time, bool) {
	if extensions >= 5 {
		return endsAt, false
	}
	if !now.Before(endsAt) {
		return endsAt, false
	}
	if endsAt.Sub(now) <= 60*time.Second {
		return endsAt.Add(60 * time.Second), true
	}
	return endsAt, false
}
