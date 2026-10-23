package app

import (
	"context"
	"log/slog"
	"time"

	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
)

type Scheduler struct {
	auctions service.AuctionService
	rooms    *realtime.RoomManager
	interval time.Duration
	now      func() time.Time
}

func NewScheduler(
	auctions service.AuctionService,
	rooms *realtime.RoomManager,
	interval time.Duration,
	now func() time.Time,
) *Scheduler {
	if interval <= 0 {
		interval = time.Second
	}
	if now == nil {
		now = time.Now
	}

	return &Scheduler{
		auctions: auctions,
		rooms:    rooms,
		interval: interval,
		now:      now,
	}
}

func (s *Scheduler) CloseDue(ctx context.Context) error {
	ended, err := s.auctions.CloseDue(ctx)
	if err != nil {
		return err
	}

	for _, auction := range ended {
		s.rooms.PublishEnded(auction)
	}

	if len(ended) > 0 {
		slog.Info(
			"scheduler closed due auctions",
			"count", len(ended),
			"at", s.now().UTC(),
		)
	}

	return nil
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CloseDue(ctx); err != nil {
				slog.Error("scheduler close due", "error", err)
			}
		}
	}
}
