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
	r.Get("/r/{token}", handler.trackClick)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/affiliate/links", handler.createReferralLink)
			r.Get("/affiliate/dashboard", handler.getDashboard)
			r.Get("/affiliate/earnings", handler.listEarnings)
			r.Post("/affiliate/exports", handler.createExport)
			r.Post("/admin/affiliates/{affiliate_id}/suspend", handler.suspendAffiliate)
			r.Post("/admin/affiliates/{affiliate_id}/attributions", handler.manualAttribution)
		})
	})
	return r
}
