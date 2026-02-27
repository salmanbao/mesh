package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/application"
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
		r.Get("/social/health", handler.getHealth)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/social/accounts/connect", handler.connectAccount)
			r.Get("/social/accounts", handler.listAccounts)
			r.Post("/social/posts/validate", handler.validatePost)
		})
	})

	return r
}
