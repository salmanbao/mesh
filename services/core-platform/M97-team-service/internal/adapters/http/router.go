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
		r.Use(authMiddleware)
		r.Post("/team", handler.createTeam)
		r.Get("/team/membership", handler.checkMembership)
		r.Post("/team/invites/{invite_id}/accept", handler.acceptInvite)
		r.Get("/team/{team_id}", handler.getTeam)
		r.Post("/team/{team_id}/invites", handler.createInvite)
	})
	return r
}
