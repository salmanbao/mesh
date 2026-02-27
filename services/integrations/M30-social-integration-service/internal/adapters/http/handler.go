package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) connectAccount(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ConnectAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ConnectAccount(r.Context(), actor, application.ConnectAccountInput{
		UserID:    req.UserID,
		Platform:  req.Platform,
		Handle:    req.Handle,
		OAuthCode: req.OAuthCode,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.ConnectAccountResponse{
		SocialAccountID: row.SocialAccountID,
		UserID:          row.UserID,
		Platform:        row.Platform,
		Handle:          row.Handle,
		Status:          row.Status,
		ConnectedAt:     row.ConnectedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	rows, err := h.service.ListAccounts(r.Context(), actor, r.URL.Query().Get("user_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	items := make([]contracts.SocialAccountItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.SocialAccountItem{
			SocialAccountID: row.SocialAccountID,
			UserID:          row.UserID,
			Platform:        row.Platform,
			Handle:          row.Handle,
			Status:          row.Status,
			ConnectedAt:     row.ConnectedAt.UTC().Format(time.RFC3339),
			Source:          row.Source,
		})
	}
	writeSuccess(w, http.StatusOK, "", contracts.ListAccountsResponse{Accounts: items})
}

func (h *Handler) validatePost(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ValidatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ValidatePost(r.Context(), actor, application.ValidatePostInput{
		UserID:   req.UserID,
		Platform: req.Platform,
		PostID:   req.PostID,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.ValidatePostResponse{
		ValidationID: row.ValidationID,
		UserID:       row.UserID,
		Platform:     row.Platform,
		PostID:       row.PostID,
		IsValid:      row.IsValid,
		Reason:       row.Reason,
		ValidatedAt:  row.ValidatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) getHealth(w http.ResponseWriter, r *http.Request) {
	health, err := h.service.GetHealth(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	status := http.StatusOK
	if health.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}
	writeSuccess(w, status, "", health)
}
