package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/application"
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
		r.Get("/api/v1/config", handler.getConfig)
		r.Patch("/api/v1/config/{key}", handler.patchConfig)
		r.Post("/api/v1/config/import", handler.importConfig)
		r.Get("/api/v1/config/export", handler.exportConfig)
		r.Post("/api/v1/config/rollback", handler.rollbackConfig)
		r.Get("/api/v1/config/audit", handler.queryAudit)
		r.Post("/api/v1/config/rollout-rules", handler.upsertRolloutRule)
	})
	return r
}
