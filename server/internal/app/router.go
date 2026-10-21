package app

import (
	"net/http"

	"live-auction/server/internal/controller"
)

func NewRouter(health *controller.HealthController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Health)
	return mux
}
