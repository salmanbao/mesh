package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/application"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }
func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ok", nil) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { writeSuccess(w, http.StatusOK, "ready", nil) })
	r.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/referral-fraud/score", handler.scoreReferral)
			r.Get("/referral-fraud/decisions/{event_id}", handler.getDecision)
			r.Post("/referral-fraud/disputes", handler.submitDispute)
			r.Get("/referral-fraud/metrics", handler.getMetrics)
		})
	})
	return r
}
