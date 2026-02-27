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
			r.Post("/tracking/posts/validate", handler.validatePost)
			r.Post("/tracking/posts", handler.registerPost)
			r.Get("/tracking/posts/{id}", handler.getPost)
			r.Get("/tracking/posts/{id}/metrics", handler.getMetrics)
		})
	})
	return r
}
