package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/ports"
)

func (h *Handler) requestPayout(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.RequestPayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	payout, err := h.service.RequestPayout(r.Context(), actor, application.RequestPayoutInput{
		UserID:       strings.TrimSpace(req.UserID),
		SubmissionID: strings.TrimSpace(req.SubmissionID),
		Amount:       req.Amount,
		Currency:     strings.TrimSpace(req.Currency),
		Method:       domain.PayoutMethod(strings.ToLower(strings.TrimSpace(req.Method))),
		ScheduledAt:  req.ScheduledAt,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", payout)
}

func (h *Handler) getPayout(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	payoutID := chi.URLParam(r, "id")
	payout, err := h.service.GetPayout(r.Context(), actor, payoutID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", payout)
}

func (h *Handler) listHistory(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	query := ports.HistoryQuery{
		UserID: strings.TrimSpace(r.URL.Query().Get("user_id")),
		Limit:  parseIntOrDefault(r.URL.Query().Get("limit"), 20),
		Offset: parseIntOrDefault(r.URL.Query().Get("offset"), 0),
	}
	out, err := h.service.ListPayoutHistory(r.Context(), actor, query)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"items":      out.Items,
		"pagination": out.Pagination,
	})
}

func parseIntOrDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
