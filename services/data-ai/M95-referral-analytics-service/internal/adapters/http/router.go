package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/application"
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
			r.Get("/referral-analytics/funnel", handler.getFunnel)
			r.Get("/referral-analytics/leaderboard", handler.getLeaderboard)
			r.Get("/referral-analytics/cohorts/retention", handler.getCohortRetention)
			r.Get("/referral-analytics/geo", handler.getGeo)
			r.Get("/referral-analytics/forecast", handler.getForecast)
			r.Post("/referral-analytics/exports", handler.createExport)
			r.Get("/referral-analytics/exports/{id}", handler.getExport)
		})
	})
	return r
}
