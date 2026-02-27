package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/application"
)

func NewRouter(handler *Handler, service *application.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Use(metricsMiddleware(service))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "", map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, "", map[string]string{"status": "ready"})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Get("/cache/health", handler.getHealth)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/cache/{key}", handler.getCache)
			r.Put("/cache/{key}", handler.putCache)
			r.Delete("/cache/{key}", handler.deleteCache)
			r.Post("/cache/invalidate", handler.invalidateCache)
			r.Get("/cache/metrics", handler.getMetrics)
		})
	})
	return r
}
