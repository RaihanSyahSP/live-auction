package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/app"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
)

type fakeSchedulerService struct {
	service.AuctionService
	closeDueResult []auction_response.AuctionResponse
	closeDueErr    error
	closeDueCalls  int
}

func (f *fakeSchedulerService) CloseDue(
	ctx context.Context,
) ([]auction_response.AuctionResponse, error) {
	f.closeDueCalls++
	return f.closeDueResult, f.closeDueErr
}

func receiveSchedulerEvent(t *testing.T, client *realtime.Client) realtime.Event {
	t.Helper()

	select {
	case data, ok := <-client.Send:
		if !ok {
			t.Fatal("client channel closed")
		}

		var event realtime.Event
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return realtime.Event{}
	}
}

func TestSchedulerCloseDuePublishesEnded(t *testing.T) {
	auctionID := uuid.New()
	rooms := realtime.NewRoomManager()
	client, _ := rooms.Join(auctionID, nil, realtime.Snapshot{
		Version: 1,
		Auction: auction_response.AuctionResponse{
			ID:      auctionID,
			Status:  domain.AuctionStatusLive,
			Version: 1,
		},
	})

	for len(client.Send) > 0 {
		<-client.Send
	}

	svc := &fakeSchedulerService{
		closeDueResult: []auction_response.AuctionResponse{{
			ID:      auctionID,
			Status:  domain.AuctionStatusEnded,
			Version: 2,
		}},
	}
	scheduler := app.NewScheduler(svc, rooms, time.Second, time.Now)

	if err := scheduler.CloseDue(context.Background()); err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}

	event := receiveSchedulerEvent(t, client)
	if event.Type != "ended" {
		t.Fatalf("event type = %q, want ended", event.Type)
	}
	if event.Version != 2 {
		t.Errorf("event version = %d, want 2", event.Version)
	}
}

func TestSchedulerCloseDueIgnoresDuplicateEnded(t *testing.T) {
	auctionID := uuid.New()
	rooms := realtime.NewRoomManager()
	client, _ := rooms.Join(auctionID, nil, realtime.Snapshot{
		Version: 1,
		Auction: auction_response.AuctionResponse{
			ID:      auctionID,
			Status:  domain.AuctionStatusLive,
			Version: 1,
		},
	})

	for len(client.Send) > 0 {
		<-client.Send
	}

	ended := auction_response.AuctionResponse{
		ID:      auctionID,
		Status:  domain.AuctionStatusEnded,
		Version: 2,
	}
	svc := &fakeSchedulerService{
		closeDueResult: []auction_response.AuctionResponse{ended},
	}
	scheduler := app.NewScheduler(svc, rooms, time.Second, time.Now)

	if err := scheduler.CloseDue(context.Background()); err != nil {
		t.Fatalf("first CloseDue() error = %v", err)
	}
	receiveSchedulerEvent(t, client)

	if err := scheduler.CloseDue(context.Background()); err != nil {
		t.Fatalf("second CloseDue() error = %v", err)
	}
	if len(client.Send) != 0 {
		t.Fatal("duplicate ended event was published")
	}
}

func TestSchedulerCloseDueReturnsServiceError(t *testing.T) {
	svc := &fakeSchedulerService{closeDueErr: errors.New("database down")}
	scheduler := app.NewScheduler(svc, realtime.NewRoomManager(), time.Second, time.Now)

	if err := scheduler.CloseDue(context.Background()); err == nil {
		t.Fatal("CloseDue() error = nil, want service error")
	}
}

func TestSchedulerRunStopsOnContextCancel(t *testing.T) {
	svc := &fakeSchedulerService{}
	scheduler := app.NewScheduler(svc, realtime.NewRoomManager(), time.Hour, time.Now)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}
