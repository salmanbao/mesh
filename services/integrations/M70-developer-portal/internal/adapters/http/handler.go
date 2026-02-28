package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/application"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) registerDeveloper(w http.ResponseWriter, r *http.Request) {
	var req contracts.RegisterDeveloperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	developer, session, err := h.service.RegisterDeveloper(r.Context(), actorFromContext(r.Context()), application.RegisterDeveloperInput{
		Email:   req.Email,
		AppName: req.AppName,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "developer registered", contracts.RegisterDeveloperResponse{
		Developer: toDeveloperResponse(developer),
		Session:   toSessionResponse(session),
	})
}

func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateAPIKey(r.Context(), actorFromContext(r.Context()), application.CreateAPIKeyInput{
		DeveloperID: req.DeveloperID,
		Label:       req.Label,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "api key created", toAPIKeyResponse(row))
}

func (h *Handler) rotateAPIKey(w http.ResponseWriter, r *http.Request) {
	rotation, oldKey, newKey, err := h.service.RotateAPIKey(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "api key rotated", contracts.APIKeyRotationResponse{
		RotationID:  rotation.RotationID,
		OldKey:      toAPIKeyResponse(oldKey),
		NewKey:      toAPIKeyResponse(newKey),
		DeveloperID: rotation.DeveloperID,
		CreatedAt:   rotation.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.RevokeAPIKey(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "api key revoked", toAPIKeyResponse(row))
}

func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateWebhook(r.Context(), actorFromContext(r.Context()), application.CreateWebhookInput{
		DeveloperID: req.DeveloperID,
		URL:         req.URL,
		EventType:   req.EventType,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "webhook created", toWebhookResponse(row))
}

func (h *Handler) testWebhook(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.TestWebhook(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "webhook test delivered", toDeliveryResponse(row))
}

func toDeveloperResponse(row domain.Developer) contracts.DeveloperResponse {
	return contracts.DeveloperResponse{
		DeveloperID: row.DeveloperID,
		Email:       row.Email,
		AppName:     row.AppName,
		Tier:        row.Tier,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toSessionResponse(row domain.DeveloperSession) contracts.SessionResponse {
	return contracts.SessionResponse{
		SessionID:    row.SessionID,
		DeveloperID:  row.DeveloperID,
		SessionToken: row.SessionToken,
		Status:       row.Status,
		ExpiresAt:    row.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:    row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toAPIKeyResponse(row domain.APIKey) contracts.APIKeyResponse {
	out := contracts.APIKeyResponse{
		KeyID:       row.KeyID,
		DeveloperID: row.DeveloperID,
		Label:       row.Label,
		MaskedKey:   row.MaskedKey,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.RevokedAt != nil {
		out.RevokedAt = row.RevokedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toWebhookResponse(row domain.Webhook) contracts.WebhookResponse {
	return contracts.WebhookResponse{
		WebhookID:   row.WebhookID,
		DeveloperID: row.DeveloperID,
		URL:         row.URL,
		EventType:   row.EventType,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toDeliveryResponse(row domain.WebhookDelivery) contracts.WebhookDeliveryResponse {
	return contracts.WebhookDeliveryResponse{
		DeliveryID: row.DeliveryID,
		WebhookID:  row.WebhookID,
		Status:     row.Status,
		TestEvent:  row.TestEvent,
		CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
	}
}
