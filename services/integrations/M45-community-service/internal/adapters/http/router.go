package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ok", nil) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ready", nil) })
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/community/integrations", handler.connectIntegration)
			r.Get("/community/integrations/{integration_id}", handler.getIntegration)
			r.Get("/community/integrations/{integration_id}/health", handler.getIntegrationHealth)
			r.Post("/admin/community/grants", handler.manualGrant)
			r.Get("/admin/community/grants/{grant_id}", handler.getGrant)
			r.Get("/admin/audit-logs", handler.listAuditLogs)
		})
	})
	return r
}
