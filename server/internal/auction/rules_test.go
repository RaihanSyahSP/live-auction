package auction

import (
	"testing"
	"time"

	"live-auction/server/internal/model/domain"

	"github.com/google/uuid"
)

func TestMinimumBid(t *testing.T) {
	auction := domain.Auction{
		StartingPrice: 1000,
		MinIncrement:  100,
	}
	if got := MinimumBid(auction); got != 1000 {
		t.Fatalf("minimum = %d, want 1000", got)
	}

	bidder := uuid.New()
	auction.HighestBidderID = &bidder
	auction.CurrentPrice = 1500
	if got := MinimumBid(auction); got != 1600 {
		t.Fatalf("minimum = %d, want 1600", got)
	}
}

func TestNextEndsAt(t *testing.T) {
	now := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		remaining  time.Duration
		extensions int
		want       bool
	}{
		{name: "outside window", remaining: 61 * time.Second, want: false},
		{name: "boundary 60 seconds", remaining: 60 * time.Second, want: true},
		{name: "inside window", remaining: 30 * time.Second, want: true},
		{name: "already ended", remaining: 0, want: false},
		{name: "maximum extensions", remaining: 30 * time.Second, extensions: 5, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endsAt := now.Add(tt.remaining)
			got, extended := NextEndsAt(now, endsAt, tt.extensions)
			if extended != tt.want {
				t.Fatalf("extended = %v, want %v", extended, tt.want)
			}

			if extended && !got.Equal(endsAt.Add(60*time.Second)) {
				t.Errorf("new endsAt = %v, want %v", got, endsAt.Add(60*time.Second))
			}
			if !extended && !got.Equal(endsAt) {
				t.Errorf("endsAt changed to %v, want %v", got, endsAt)
			}
		})
	}
}
