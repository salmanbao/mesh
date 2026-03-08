package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) getConsent(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.PathValue("user_id"))
	row, err := h.service.GetConsent(r.Context(), actorFromContext(r.Context()), userID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "consent record", toConsentResponse(row))
}

func (h *Handler) updateConsent(w http.ResponseWriter, r *http.Request) {
	var req contracts.UpdateConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}

	row, err := h.service.UpdateConsent(r.Context(), actorFromContext(r.Context()), application.UpdateConsentInput{
		UserID:      strings.TrimSpace(r.PathValue("user_id")),
		Preferences: req.Preferences,
		Reason:      req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "consent updated", toConsentResponse(row))
}

func (h *Handler) withdrawConsent(w http.ResponseWriter, r *http.Request) {
	var req contracts.WithdrawConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}

	row, err := h.service.WithdrawConsent(r.Context(), actorFromContext(r.Context()), application.WithdrawConsentInput{
		UserID:   strings.TrimSpace(r.PathValue("user_id")),
		Category: req.Category,
		Reason:   req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "consent withdrawn", toConsentResponse(row))
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	rows, err := h.service.ListHistory(
		r.Context(),
		actorFromContext(r.Context()),
		strings.TrimSpace(r.PathValue("user_id")),
		limit,
	)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}

	resp := contracts.ConsentHistoryResponse{Items: make([]contracts.ConsentHistoryEntryResponse, 0, len(rows))}
	for _, row := range rows {
		resp.Items = append(resp.Items, toConsentHistoryResponse(row))
	}
	writeSuccess(w, http.StatusOK, "consent history", resp)
}

func toConsentResponse(row domain.ConsentRecord) contracts.ConsentRecordResponse {
	return contracts.ConsentRecordResponse{
		UserID:      row.UserID,
		Preferences: row.Preferences,
		Status:      row.Status,
		UpdatedAt:   row.UpdatedAt.UTC().Format(time.RFC3339),
		UpdatedBy:   row.UpdatedBy,
	}
}

func toConsentHistoryResponse(row domain.ConsentHistory) contracts.ConsentHistoryEntryResponse {
	return contracts.ConsentHistoryEntryResponse{
		EventID:    row.EventID,
		EventType:  row.EventType,
		UserID:     row.UserID,
		Category:   row.Category,
		Reason:     row.Reason,
		ChangedBy:  row.ChangedBy,
		OccurredAt: row.OccurredAt.UTC().Format(time.RFC3339),
	}
}
