package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

func (h *Handler) oidcAuthorize(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	redirectURI := r.URL.Query().Get("redirect_uri")
	clientContext := r.URL.Query().Get("client_context")
	loginHint := r.URL.Query().Get("login_hint")
	ipAddress := readIP(r)

	res, err := h.service.OIDCAuthorize(r.Context(), provider, redirectURI, clientContext, loginHint, ipAddress)
	if err != nil {
		writeMappedError(r.Context(), w, "oidc_authorize", err)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("response_mode"), "json") {
		writeSuccess(w, http.StatusOK, res)
		return
	}
	http.Redirect(w, r, res.AuthorizeURL, http.StatusFound)
}

func (h *Handler) oidcCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	result, err := h.service.OIDCCallback(r.Context(), code, state)
	if err != nil {
		writeMappedError(r.Context(), w, "oidc_callback", err)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("response_mode"), "json") {
		writeSuccess(w, http.StatusOK, result)
		return
	}
	redirectURL := result.RedirectURL
	if strings.TrimSpace(redirectURL) == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) oidcLink(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "oidc_link")
		return
	}
	var req application.OIDCLinkRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "oidc_link", err)
		return
	}
	if err := h.service.LinkOIDC(r.Context(), token, req); err != nil {
		writeMappedError(r.Context(), w, "oidc_link", err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"provider": req.Provider,
		"linked":   true,
	})
}

func (h *Handler) registerComplete(w http.ResponseWriter, r *http.Request) {
	var req application.RegisterCompleteRequest
	if err := decodeBody(r, &req); err != nil {
		writeValidationError(r.Context(), w, "register_complete", err)
		return
	}
	res, err := h.service.RegisterComplete(r.Context(), req)
	if err != nil {
		writeMappedError(r.Context(), w, "register_complete", err)
		return
	}
	writeSuccess(w, http.StatusOK, res)
}

func (h *Handler) oidcUnlink(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenFromContext(r)
	if !ok {
		writeMissingBearerError(r.Context(), w, "oidc_unlink")
		return
	}
	provider := chi.URLParam(r, "provider")
	if err := h.service.UnlinkOIDC(r.Context(), token, provider); err != nil {
		writeMappedError(r.Context(), w, "oidc_unlink", err)
		return
	}
	writeMessage(w, http.StatusOK, "OIDC connection removed")
}
