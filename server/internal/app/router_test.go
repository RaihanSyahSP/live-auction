package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"live-auction/server/internal/app"
	"live-auction/server/internal/config"
	"live-auction/server/internal/controller"
)

type routerHealthChecker struct{}

func (routerHealthChecker) Ping(ctx context.Context) error {
	return nil
}

func TestRouterHealthz(t *testing.T) {
	router := app.NewRouter(app.RouterDeps{
		Health: controller.NewHealthController(routerHealthChecker{}),
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
}

func TestRouterMetrics(t *testing.T) {
	metricsCalled := false
	router := app.NewRouter(app.RouterDeps{
		Metrics: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metricsCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !metricsCalled {
		t.Error("metrics handler was not called")
	}
}

func TestNewServer(t *testing.T) {
	handler := http.NewServeMux()
	server := app.NewServer(config.Config{Port: "9090"}, handler)

	if server.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", server.Addr)
	}
	if server.Handler != handler {
		t.Error("Handler was not set")
	}
	if server.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout must be greater than zero")
	}
	if server.IdleTimeout <= 0 {
		t.Error("IdleTimeout must be greater than zero")
	}
}
