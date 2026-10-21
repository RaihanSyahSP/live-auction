package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"live-auction/server/internal/app"
	"live-auction/server/internal/config"
	"live-auction/server/internal/controller"
)

func TestRouterHealthz(t *testing.T) {
	router := app.NewRouter(controller.NewHealthController())

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
