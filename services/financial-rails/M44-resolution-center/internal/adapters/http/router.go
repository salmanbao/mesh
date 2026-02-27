package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/application"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ok", nil) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ready", nil) })

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/disputes", handler.createDispute)
			r.Get("/disputes/{dispute_id}", handler.getDispute)
			r.Post("/disputes/{dispute_id}/messages", handler.sendMessage)
			r.Post("/admin/disputes/{dispute_id}/approve", handler.approveDispute)
		})
	})
	return r
}
