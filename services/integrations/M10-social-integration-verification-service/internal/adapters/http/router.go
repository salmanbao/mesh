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
			r.Post("/social/connect/{provider}", handler.connect)
			r.Post("/social/callback/{provider}", handler.callback)
			r.Get("/social/accounts", handler.listAccounts)
			r.Delete("/social/accounts/{social_account_id}", handler.disconnect)
			r.Post("/social/accounts/{social_account_id}/followers-sync", handler.followersSync)
			r.Post("/social/posts/validate", handler.validatePost)
			r.Post("/social/posts/compliance-violation", handler.complianceViolation)
		})
	})
	return r
}
