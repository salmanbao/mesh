package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Get("/health", handler.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/webhooks", handler.createWebhook)
		r.Get("/webhooks/{webhook_id}", handler.getWebhook)
		r.Post("/webhooks/{webhook_id}/test", handler.testWebhook)
		r.Get("/webhooks/{webhook_id}/deliveries", handler.listDeliveries)
		r.Get("/webhooks/{webhook_id}/analytics", handler.analytics)
		r.Post("/webhooks/{webhook_id}/enable", handler.enableWebhook)
	})
	return r
}
