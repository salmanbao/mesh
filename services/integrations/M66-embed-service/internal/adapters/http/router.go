package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Get("/embed/{entity_type}/{entity_id}", handler.renderEmbed)
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/embeds/{entity_type}/{entity_id}/settings", handler.getSettings)
			r.Post("/embeds/{entity_type}/{entity_id}/settings", handler.postSettings)
			r.Get("/embeds/{entity_type}/{entity_id}/analytics", handler.getAnalytics)
		})
	})
	return r
}
