package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/application"
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
		r.Post("/v1/storage/policies", handler.createPolicy)
		r.Get("/v1/storage/analytics/summary", handler.analyticsSummary)
		r.Post("/storage/move-to-glacier", handler.moveToGlacier)
		r.Post("/storage/schedule-deletion", handler.scheduleDeletion)
		r.Get("/storage/audit/deletions", handler.queryDeletionAudit)
	})
	return r
}
