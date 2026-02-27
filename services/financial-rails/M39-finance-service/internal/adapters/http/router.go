package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/application"
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
			r.Post("/webhooks/provider", handler.providerWebhook)
		})

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/transactions", handler.createTransaction)
			r.Get("/transactions/{id}", handler.getTransaction)
			r.Get("/transactions", handler.listTransactions)
			r.Get("/balances/{userID}", handler.getBalance)
			r.Post("/refunds", handler.createRefund)
		})
	})
	return r
}
