package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) connectIntegration(w http.ResponseWriter, r *http.Request) {
	var req contracts.ConnectIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ConnectIntegration(r.Context(), actorFromContext(r.Context()), application.ConnectIntegrationInput{Platform: req.Platform, CommunityName: req.CommunityName, Config: req.Config})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "integration connected", toIntegrationResponse(row))
}
func (h *Handler) getIntegration(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.GetIntegration(r.Context(), actorFromContext(r.Context()), chi.URLParam(r, "integration_id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "integration", toIntegrationResponse(row))
}
func (h *Handler) getIntegrationHealth(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.GetIntegrationHealth(r.Context(), actorFromContext(r.Context()), chi.URLParam(r, "integration_id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "integration health", contracts.HealthCheckResponse{HealthCheckID: row.HealthCheckID, IntegrationID: row.IntegrationID, Status: string(row.Status), CheckedAt: row.CheckedAt.UTC().Format(time.RFC3339), LatencyMS: row.LatencyMS, HTTPStatusCode: row.HTTPStatusCode})
}
func (h *Handler) manualGrant(w http.ResponseWriter, r *http.Request) {
	var req contracts.ManualGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateManualGrant(r.Context(), actorFromContext(r.Context()), application.ManualGrantInput{UserID: req.UserID, ProductID: req.ProductID, IntegrationID: req.IntegrationID, Reason: req.Reason, Tier: req.Tier})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "manual grant created", toGrantResponse(row))
}
func (h *Handler) getGrant(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.GetGrant(r.Context(), actorFromContext(r.Context()), chi.URLParam(r, "grant_id"))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "grant", toGrantResponse(row))
}
func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	var fromPtr, toPtr *time.Time
	if raw := r.URL.Query().Get("from"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			fromPtr = &t
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			toPtr = &t
		}
	}
	rows, err := h.service.ListAuditLogs(r.Context(), actorFromContext(r.Context()), r.URL.Query().Get("user_id"), fromPtr, toPtr)
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.AuditLogListResponse{Items: make([]contracts.AuditLogResponse, 0, len(rows))}
	for _, row := range rows {
		resp.Items = append(resp.Items, contracts.AuditLogResponse{AuditLogID: row.AuditLogID, Timestamp: row.Timestamp.UTC().Format(time.RFC3339), ActionType: row.ActionType, PerformedBy: row.PerformedBy, PerformerRole: row.PerformerRole, UserID: row.UserID, IntegrationID: row.IntegrationID, ProductID: row.ProductID, GrantID: row.GrantID, Reason: row.Reason, Outcome: row.Outcome, Metadata: row.Metadata})
	}
	writeSuccess(w, http.StatusOK, "audit logs", resp)
}
func toIntegrationResponse(row domain.CommunityIntegration) contracts.CommunityIntegrationResponse {
	return contracts.CommunityIntegrationResponse{IntegrationID: row.IntegrationID, CreatorID: row.CreatorID, Platform: row.Platform, Status: string(row.Status), CommunityName: row.CommunityName, Config: row.Config, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)}
}
func toGrantResponse(row domain.CommunityGrant) contracts.CommunityGrantResponse {
	return contracts.CommunityGrantResponse{GrantID: row.GrantID, Status: string(row.Status), UserID: row.UserID, ProductID: row.ProductID, IntegrationID: row.IntegrationID, Tier: row.Tier, GrantedAt: row.GrantedAt.UTC().Format(time.RFC3339)}
}
