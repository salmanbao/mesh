package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/application"
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
			r.Get("/seller/risk-dashboard", handler.getSellerRiskDashboard)
			r.Post("/disputes", handler.fileDispute)
			r.Post("/disputes/{dispute_id}/evidence", handler.submitDisputeEvidence)
		})
		r.Post("/webhooks/chargeback", handler.handleChargebackWebhook)
	})

	return r
}
