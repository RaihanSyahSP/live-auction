package controller_test

import (
	"encoding/json"
	"testing"
	"time"

	"live-auction/server/internal/realtime"
)

func drainRealtimeClient(client *realtime.Client) {
	for {
		select {
		case <-client.Send:
		default:
			return
		}
	}
}

func receiveRealtimeEvent(t *testing.T, client *realtime.Client) realtime.Event {
	t.Helper()

	select {
	case data, ok := <-client.Send:
		if !ok {
			t.Fatal("realtime client channel closed")
		}

		var event realtime.Event
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatalf("decode realtime event: %v", err)
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for realtime event")
		return realtime.Event{}
	}
}
