package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/application"
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
	r.Use(recoverMiddleware)
	r.Use(loggingMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { writeMessage(w, http.StatusOK, "ok") })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { writeMessage(w, http.StatusOK, "ready") })
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/", handler.swaggerUI)
	r.Get("/swagger/openapi.yaml", handler.swaggerSpec)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/profiles", func(r chi.Router) {
			r.Get("/{username}", handler.getPublicProfile)
			r.Get("/username-availability", handler.checkUsernameAvailability)

			r.Group(func(r chi.Router) {
				r.Use(handler.authMiddleware)
				r.Get("/me", handler.getMyProfile)
				r.Put("/me", handler.updateMyProfile)
				r.Put("/me/username", handler.changeUsername)
				r.Post("/me/avatar", handler.uploadAvatar)
				r.Post("/me/social-links", handler.addSocialLink)
				r.Delete("/me/social-links/{platform}", handler.deleteSocialLink)
				r.Post("/me/payout-methods", handler.putPayoutMethod)
				r.Put("/me/payout-methods/{method_type}", handler.updatePayoutMethod)
				r.Post("/me/kyc/documents", handler.uploadKYCDocument)
				r.Get("/me/kyc/status", handler.getKYCStatus)
			})
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(handler.authMiddleware)
			r.Get("/profiles", handler.adminListProfiles)
			r.Get("/kyc/queue", handler.adminListProfiles)
			r.Post("/kyc/{user_id}/approve", handler.adminApproveKYC)
			r.Post("/kyc/{user_id}/reject", handler.adminRejectKYC)
		})
	})
	return r
}
