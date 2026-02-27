package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/application"
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
	r.Get("/health", handler.getHealth)
	r.Get("/metrics", handler.getMetrics)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/alert-rules", handler.createAlertRule)
		r.Get("/alert-rules", handler.listAlertRules)
		r.Get("/incidents", handler.listIncidents)
		r.Post("/silences", handler.createSilence)
		r.Get("/audit", handler.queryAudit)
	})

	return r
}
