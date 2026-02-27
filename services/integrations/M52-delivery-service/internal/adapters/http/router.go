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
	r.Get("/download/{token}", handler.downloadByToken)
	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/delivery/download-link/{product_id}", handler.getDownloadLink)
			r.Put("/admin/delivery/products/{product_id}/file", handler.upsertProductFile)
			r.Post("/admin/delivery/revoke-links", handler.revokeLinks)
		})
	})
	return r
}
