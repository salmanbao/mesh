package http

import (
	"net/http"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

func (h *Handler) passwordResetRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "password_reset_request", err)
		return
	}
	if err := h.service.RequestPasswordReset(r.Context(), req.Email); err != nil {
		writeMappedError(r.Context(), w, "password_reset_request", err)
		return
	}
	writeMessage(w, http.StatusOK, "If the email exists, a password reset link has been sent")
}

func (h *Handler) passwordReset(w http.ResponseWriter, r *http.Request) {
	var req application.PasswordResetRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "password_reset", err)
		return
	}
	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		writeMappedError(r.Context(), w, "password_reset", err)
		return
	}
	writeMessage(w, http.StatusOK, "Password reset successful. You can now login with your new password.")
}

func (h *Handler) emailVerifyRequest(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "email_verify_request")
		return
	}
	if err := h.service.RequestEmailVerification(r.Context(), token); err != nil {
		writeMappedError(r.Context(), w, "email_verify_request", err)
		return
	}
	writeMessage(w, http.StatusOK, "Verification email sent")
}

func (h *Handler) emailVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "email_verify", err)
		return
	}
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		writeMappedError(r.Context(), w, "email_verify", err)
		return
	}
	writeMessage(w, http.StatusOK, "Email verified successfully")
}
