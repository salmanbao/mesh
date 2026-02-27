package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/application"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ok", nil) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ready", nil) })

	r.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/rewards/calculate", handler.calculateReward)
			r.Get("/rewards/submissions/{submission_id}", handler.getReward)
			r.Get("/rewards/rollovers/{user_id}", handler.getRollover)
			r.Get("/rewards/history", handler.listRewardHistory)
		})
	})
	return r
}
