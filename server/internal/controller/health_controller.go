package controller

import (
	"context"
	"encoding/json"
	"net/http"
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type HealthController struct {
	db HealthChecker
}

func NewHealthController(db HealthChecker) *HealthController {
	return &HealthController{db: db}
}

func (h *HealthController) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := h.db.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
