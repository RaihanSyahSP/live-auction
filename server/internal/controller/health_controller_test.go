package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"live-auction/server/internal/controller"
)

type fakeHealthChecker struct {
	err error
}

func (f *fakeHealthChecker) Ping(ctx context.Context) error {
	return f.err
}

func TestHealth(t *testing.T) {
	handler := controller.NewHealthController(&fakeHealthChecker{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("body status = %q, want ok", body["status"])
	}
}

func TestHealthReturnsServiceUnavailableWhenPingFails(t *testing.T) {
	handler := controller.NewHealthController(&fakeHealthChecker{
		err: errors.New("database unavailable"),
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "unhealthy" {
		t.Errorf("body status = %q, want unhealthy", body["status"])
	}
}
