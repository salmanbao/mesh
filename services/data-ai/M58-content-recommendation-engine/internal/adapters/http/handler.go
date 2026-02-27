package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/contracts"
)

func (h *Handler) getRecommendations(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	resp, err := h.service.GetRecommendations(r.Context(), actor, application.GetRecommendationsInput{
		Role:    strings.TrimSpace(r.URL.Query().Get("role")),
		Limit:   limit,
		Segment: strings.TrimSpace(r.URL.Query().Get("segment")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"recommendations": resp.Recommendations,
		"meta":            resp.Meta,
	})
}

func (h *Handler) recordFeedback(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required", requestIDFromContext(r.Context()))
		return
	}
	recommendationID := chi.URLParam(r, "recommendation_id")
	var req contracts.RecommendationFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	record, err := h.service.RecordFeedback(r.Context(), actor, recommendationID, application.FeedbackInput{
		EventID:          strings.TrimSpace(req.EventID),
		EventType:        strings.TrimSpace(req.EventType),
		OccurredAt:       strings.TrimSpace(req.OccurredAt),
		SourceService:    strings.TrimSpace(req.SourceService),
		TraceID:          strings.TrimSpace(req.TraceID),
		SchemaVersion:    strings.TrimSpace(req.SchemaVersion),
		PartitionKeyPath: strings.TrimSpace(req.PartitionKeyPath),
		PartitionKey:     strings.TrimSpace(req.PartitionKey),
		EntityID:         strings.TrimSpace(req.Data.EntityID),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"feedback_id": record.FeedbackID,
		"recorded_at": record.CreatedAt,
	})
}

func (h *Handler) createOverride(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required", requestIDFromContext(r.Context()))
		return
	}
	var req contracts.RecommendationOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ApplyOverride(r.Context(), actor, application.OverrideInput{
		OverrideType: strings.TrimSpace(req.OverrideType),
		EntityID:     strings.TrimSpace(req.EntityID),
		Scope:        strings.TrimSpace(req.Scope),
		ScopeValue:   strings.TrimSpace(req.ScopeValue),
		Multiplier:   req.Multiplier,
		Reason:       strings.TrimSpace(req.Reason),
		EndDate:      strings.TrimSpace(req.EndDate),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", map[string]interface{}{
		"override_id": row.OverrideID,
		"created_at":  row.CreatedAt,
	})
}

func (h *Handler) listOverrides(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	items, err := h.service.ListOverrides(r.Context(), actor, strings.TrimSpace(r.URL.Query().Get("role")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{"items": items})
}
