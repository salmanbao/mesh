package http

import (
	"net/http"
	"strings"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req application.RegisterRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "register", err)
		return
	}
	if req.IPAddress == "" {
		req.IPAddress = readIP(r)
	}

	res, err := h.service.Register(r.Context(), req, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		writeMappedError(r.Context(), w, "register", err)
		return
	}

	writeSuccess(w, http.StatusCreated, res)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req application.LoginRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "login", err)
		return
	}

	if req.IPAddress == "" {
		req.IPAddress = readIP(r)
	}
	if req.UserAgent == "" {
		req.UserAgent = r.UserAgent()
	}

	res, err := h.service.Login(r.Context(), req)
	if err != nil {
		writeMappedError(r.Context(), w, "login", err)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) twoFAVerify(w http.ResponseWriter, r *http.Request) {
	var req application.TwoFAVerifyRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "two_fa_verify", err)
		return
	}

	if req.TempToken == "" {
		token, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if err == nil {
			req.TempToken = token
		}
	}
	if req.IPAddress == "" {
		req.IPAddress = readIP(r)
	}
	if req.UserAgent == "" {
		req.UserAgent = r.UserAgent()
	}

	res, err := h.service.Verify2FA(r.Context(), req)
	if err != nil {
		writeMappedError(r.Context(), w, "two_fa_verify", err)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) twoFASetup(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "two_fa_setup")
		return
	}
	var req application.TwoFASetupRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "two_fa_setup", err)
		return
	}

	res, err := h.service.Setup2FA(r.Context(), token, req)
	if err != nil {
		writeMappedError(r.Context(), w, "two_fa_setup", err)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "refresh")
		return
	}
	res, err := h.service.Refresh(r.Context(), token)
	if err != nil {
		writeMappedError(r.Context(), w, "refresh", err)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "logout")
		return
	}
	if err := h.service.LogoutCurrentSession(r.Context(), token); err != nil {
		writeMappedError(r.Context(), w, "logout", err)
		return
	}
	writeMessage(w, http.StatusOK, "Logged out successfully")
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "delete_account")
		return
	}
	if err := h.service.DeleteAccount(r.Context(), token); err != nil {
		writeMappedError(r.Context(), w, "delete_account", err)
		return
	}
	writeMessage(w, http.StatusOK, "Account deletion requested successfully")
}
