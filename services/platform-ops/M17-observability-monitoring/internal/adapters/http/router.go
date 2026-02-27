package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/application"
)

func NewRouter(handler *Handler, service *application.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Use(metricsMiddleware(service))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Get("/health", handler.getHealth)
	r.Get("/metrics", handler.getMetrics)

	r.Route("/api/v1/observability", func(r chi.Router) {
		r.Get("/components", handler.listComponents)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Put("/components/{name}", handler.upsertComponent)
		})
	})
	return r
}
