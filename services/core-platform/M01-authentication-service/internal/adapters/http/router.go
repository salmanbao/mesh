package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

// Handler is the HTTP adapter entrypoint for auth use-cases.
// Keeping only application dependency here preserves clean adapter boundaries.
type Handler struct {
	service *application.Service
}

// NewHandler constructs an HTTP handler bound to application service.
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// NewRouter registers M01 HTTP routes and middleware stack.
// Centralizing routes here ensures consistent auth and error behavior across endpoints.
func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)
	r.Use(recoverMiddleware)
	r.Use(loggingMiddleware)

	r.Get("/healthz", handler.healthz)
	r.Get("/readyz", handler.readyz)
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/", handler.swaggerUI)
	r.Get("/swagger/openapi.yaml", handler.swaggerSpec)

	r.Route("/auth/v1", func(r chi.Router) {
		r.Post("/register", handler.register)
		r.Post("/register/complete", handler.registerComplete)
		r.Post("/login", handler.login)
		r.Post("/password/reset-request", handler.passwordResetRequest)
		r.Post("/password/reset", handler.passwordReset)
		r.Post("/2fa/verify", handler.twoFAVerify)
		r.Post("/email/verify", handler.emailVerify)
		r.Get("/oidc/authorize", handler.oidcAuthorize)
		r.Get("/oidc/callback", handler.oidcCallback)

		r.Group(func(r chi.Router) {
			r.Use(handler.authMiddleware)
			r.Post("/refresh", handler.refresh)
			r.Post("/logout", handler.logout)
			r.Delete("/account", handler.deleteAccount)
			r.Post("/2fa/setup", handler.twoFASetup)
			r.Post("/email/verify-request", handler.emailVerifyRequest)
			r.Post("/oidc/link", handler.oidcLink)
			r.Delete("/oidc/link/{provider}", handler.oidcUnlink)
			r.Get("/sessions", handler.listSessions)
			r.Delete("/sessions/{session_id}", handler.revokeSession)
			r.Delete("/sessions", handler.revokeAllSessions)
			r.Get("/login-history", handler.loginHistory)
		})
	})

	return r
}
