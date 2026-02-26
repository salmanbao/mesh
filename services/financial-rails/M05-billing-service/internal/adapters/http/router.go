package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/application"
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
			r.Post("/invoices", handler.createInvoice)
			r.Get("/invoices/{invoice_id}", handler.getInvoice)
			r.Post("/invoices/{invoice_id}/send", handler.sendInvoice)
			r.Get("/invoices/{invoice_id}/download", handler.downloadInvoice)
			r.Get("/invoices/{invoice_id}/pdf", handler.downloadInvoice)
			r.Post("/invoices/{invoice_id}/void", handler.voidInvoice)
			r.Get("/user/invoices", handler.listUserInvoices)
			r.Get("/admin/invoices", handler.searchInvoices)
			r.Get("/user/billing/export", handler.requestBillingExport)
			r.Post("/user/billing/delete-request", handler.requestBillingDelete)
			r.Post("/refunds", handler.createRefund)
		})
	})
	return r
}
