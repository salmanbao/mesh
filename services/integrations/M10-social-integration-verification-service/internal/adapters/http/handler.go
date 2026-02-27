package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) {
	var req contracts.ConnectRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	actor := actorFromContext(r.Context())
	res, err := h.service.ConnectStart(r.Context(), actor, application.ConnectInput{Provider: chi.URLParam(r, "provider"), UserID: strings.TrimSpace(req.UserID)})
	if err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "connect initialized", contracts.ConnectResponse{AuthURL: res.AuthURL, State: res.State})
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	var req contracts.CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	acc, err := h.service.OAuthCallback(r.Context(), actor, application.CallbackInput{Provider: chi.URLParam(r, "provider"), UserID: req.UserID, Code: req.Code, State: req.State, Handle: req.Handle})
	if err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "callback processed", contracts.CallbackResponse{SocialAccountID: acc.SocialAccountID, Status: acc.Status})
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	accounts, err := h.service.ListAccounts(r.Context(), actor, userID)
	if err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	out := make([]contracts.SocialAccountResponse, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, contracts.SocialAccountResponse{SocialAccountID: a.SocialAccountID, Provider: a.Provider, Handle: a.Handle, Status: a.Status, ConnectedAt: a.ConnectedAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
	}
	writeSuccess(w, http.StatusOK, "accounts", contracts.ListAccountsResponse{Accounts: out})
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	acc, err := h.service.DisconnectAccount(r.Context(), actor, chi.URLParam(r, "social_account_id"))
	if err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "account disconnected", contracts.DisconnectResponse{SocialAccountID: acc.SocialAccountID, Status: acc.Status})
}

func (h *Handler) followersSync(w http.ResponseWriter, r *http.Request) {
	var req contracts.FollowersSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	row, err := h.service.RecordFollowersSync(r.Context(), actor, application.RecordFollowersSyncInput{SocialAccountID: chi.URLParam(r, "social_account_id"), UserID: req.UserID, FollowerCount: req.FollowerCount})
	if err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "followers synced", map[string]any{"metric_id": row.MetricID, "follower_count": row.FollowerCount})
}

func (h *Handler) validatePost(w http.ResponseWriter, r *http.Request) {
	var req contracts.PostValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	if err := h.service.ValidatePost(r.Context(), actor, application.PostValidationInput{UserID: req.UserID, Platform: req.Platform, PostID: req.PostID}); err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "post validated", nil)
}

func (h *Handler) complianceViolation(w http.ResponseWriter, r *http.Request) {
	var req contracts.ComplianceViolationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	if err := h.service.ReportComplianceViolation(r.Context(), actor, application.ComplianceViolationInput{UserID: req.UserID, Platform: req.Platform, PostID: req.PostID, Reason: req.Reason}); err != nil { code, e := mapDomainError(err); writeError(w, code, e, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "compliance violation recorded", nil)
}
