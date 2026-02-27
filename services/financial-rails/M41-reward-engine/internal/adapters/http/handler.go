package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/contracts"
)

func (h *Handler) calculateReward(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CalculateRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	reward, err := h.service.CalculateReward(r.Context(), actor, application.CalculateRewardInput{
		UserID:                  strings.TrimSpace(req.UserID),
		SubmissionID:            strings.TrimSpace(req.SubmissionID),
		CampaignID:              strings.TrimSpace(req.CampaignID),
		LockedViews:             req.LockedViews,
		RatePer1K:               req.RatePer1K,
		FraudScore:              req.FraudScore,
		VerificationCompletedAt: req.VerificationCompletedAt,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", reward)
}

func (h *Handler) getReward(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	submissionID := chi.URLParam(r, "submission_id")
	reward, err := h.service.GetReward(r.Context(), actor, submissionID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", reward)
}

func (h *Handler) getRollover(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	userID := chi.URLParam(r, "user_id")
	rollover, err := h.service.GetRollover(r.Context(), actor, userID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", rollover)
}

func (h *Handler) listRewardHistory(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 20)
	offset := parseIntOrDefault(r.URL.Query().Get("offset"), 0)
	out, err := h.service.ListRewardsByUser(r.Context(), actor, userID, limit, offset)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"items": out.Items,
		"pagination": contracts.Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  out.Total,
		},
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
