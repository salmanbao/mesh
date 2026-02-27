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
	r.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/notifications", handler.listNotifications)
			r.Get("/notifications/unread-count", handler.unreadCount)
			r.Post("/notifications/{id}/read", handler.markRead)
			r.Post("/notifications/{id}/archive", handler.archive)
			r.Post("/notifications/bulk-action", handler.bulkAction)
			r.Get("/notifications/preferences", handler.getPreferences)
			r.Put("/notifications/preferences", handler.updatePreferences)
			r.Delete("/notifications/scheduled/{id}", handler.deleteScheduled)
		})
	})
	return r
}
