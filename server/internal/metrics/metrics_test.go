package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"live-auction/server/internal/metrics"
)

func TestMetricsHandlerExposesMetrics(t *testing.T) {
	m := metrics.New()
	m.ObserveBid("success", 10*time.Millisecond)
	m.ObserveBroadcast(5 * time.Millisecond)
	m.SetConnectedClients("auction-id", 3)
	m.SetLiveAuctions(1)
	m.ObserveLogin("success")
	m.ObserveRefresh("error")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, name := range []string{
		"auction_bids_total",
		"auction_bid_duration_seconds",
		"auction_broadcast_duration_seconds",
		"auction_connected_clients",
		"auction_live_auctions",
		"auth_login_total",
		"auth_refresh_total",
	} {
		if !strings.Contains(body, name) {
			t.Errorf("metrics body does not contain %s", name)
		}
	}
}

func TestNilMetricsIsSafe(t *testing.T) {
	var m *metrics.Metrics

	m.ObserveBid("success", time.Millisecond)
	m.ObserveBroadcast(time.Millisecond)
	m.SetConnectedClients("auction-id", 1)
	m.SetLiveAuctions(1)
	m.ObserveLogin("success")
	m.ObserveRefresh("success")
}
