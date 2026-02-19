package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeMessage(w, http.StatusOK, "ok")
}

func (h *Handler) readyz(w http.ResponseWriter, _ *http.Request) {
	writeMessage(w, http.StatusOK, "ready")
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}

		claims, err := h.service.ValidateToken(r.Context(), raw)
		if err != nil {
			status, code, msg := mapDomainError(err)
			writeError(w, status, code, msg)
			return
		}

		ctx := r.Context()
		ctx = contextWithToken(ctx, raw, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func contextWithToken(ctx context.Context, token string, claims any) context.Context {
	ctx = context.WithValue(ctx, ctxKeyTokenRaw, token)
	ctx = context.WithValue(ctx, ctxKeyClaims, claims)
	return ctx
}

func tokenFromContext(r *http.Request) (string, bool) {
	v := r.Context().Value(ctxKeyTokenRaw)
	token, ok := v.(string)
	return token, ok
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req application.RegisterRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	res, err := h.service.Register(r.Context(), req, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}

	writeSuccess(w, http.StatusCreated, map[string]any{
		"user_id": res.UserID,
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req application.LoginRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
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
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) twoFAVerify(w http.ResponseWriter, r *http.Request) {
	var req application.TwoFAVerifyRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
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
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) twoFASetup(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	var req application.TwoFASetupRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	res, err := h.service.Setup2FA(r.Context(), token, req)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) passwordResetRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.service.RequestPasswordReset(r.Context(), req.Email); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "If the email exists, a password reset link has been sent")
}

func (h *Handler) passwordReset(w http.ResponseWriter, r *http.Request) {
	var req application.PasswordResetRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "Password reset successful. You can now login with your new password.")
}

func (h *Handler) emailVerifyRequest(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	if err := h.service.RequestEmailVerification(r.Context(), token); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "Verification email sent")
}

func (h *Handler) emailVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "Email verified successfully")
}

func (h *Handler) oidcAuthorize(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	nonce := r.URL.Query().Get("nonce")
	loginHint := r.URL.Query().Get("login_hint")

	redirectURL, err := h.service.OIDCAuthorize(r.Context(), provider, redirectURI, state, nonce, loginHint)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) oidcCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	redirectURL, err := h.service.OIDCCallback(r.Context(), code, state)
	if err != nil {
		status, codeE, msg := mapDomainError(err)
		writeError(w, status, codeE, msg)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) oidcLink(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	var req application.OIDCLinkRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.service.LinkOIDC(r.Context(), token, req); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"provider": req.Provider,
		"linked":   true,
	})
}

func (h *Handler) oidcUnlink(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	provider := chi.URLParam(r, "provider")
	if err := h.service.UnlinkOIDC(r.Context(), token, provider); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "OIDC connection removed")
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	res, err := h.service.Refresh(r.Context(), token)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	if err := h.service.LogoutCurrentSession(r.Context(), token); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "Logged out successfully")
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	items, err := h.service.ListSessions(r.Context(), token)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"sessions": items})
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid session_id")
		return
	}
	if err := h.service.RevokeSessionByID(r.Context(), token, sessionID); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "Session revoked successfully")
}

func (h *Handler) revokeAllSessions(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}
	if err := h.service.LogoutAllSessions(r.Context(), token); err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeMessage(w, http.StatusOK, "All sessions revoked successfully")
}

func (h *Handler) loginHistory(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return
	}

	query := application.LoginHistoryQuery{
		Page:   parseIntDefault(r.URL.Query().Get("page"), 1),
		Limit:  parseIntDefault(r.URL.Query().Get("limit"), 20),
		Days:   parseIntDefault(r.URL.Query().Get("days"), 0),
		Status: r.URL.Query().Get("status"),
	}
	items, err := h.service.ListLoginHistory(r.Context(), token, query)
	if err != nil {
		status, code, msg := mapDomainError(err)
		writeError(w, status, code, msg)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"attempts": items})
}

func decodeBody(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON value")
	}
	return nil
}

func parseIntDefault(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func readIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}
