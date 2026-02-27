package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/contracts"
)

func (h *Handler) scoreReferral(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ScoreReferralRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	dec, err := h.service.ScoreReferral(r.Context(), actor, application.ScoreInput{EventID: strings.TrimSpace(req.EventID), EventType: "affiliate.click.tracked", AffiliateID: strings.TrimSpace(req.AffiliateID), ReferralToken: strings.TrimSpace(req.ReferralToken), ReferrerID: strings.TrimSpace(req.ReferrerID), UserID: strings.TrimSpace(req.UserID), ClickIP: strings.TrimSpace(req.ClickIP), UserAgent: strings.TrimSpace(req.UserAgent), DeviceFingerprintHash: strings.TrimSpace(req.DeviceFingerprintHash), FormFillTimeMS: req.FormFillTimeMS, MouseMovementCount: req.MouseMovementCount, KeyboardCPS: req.KeyboardCPS, Amount: req.Amount, Region: strings.TrimSpace(req.Region), CampaignType: strings.TrimSpace(req.CampaignType), OccurredAt: strings.TrimSpace(req.OccurredAt), Metadata: req.Metadata, TraceID: actor.RequestID})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", dec)
}

func (h *Handler) getDecision(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	dec, err := h.service.GetDecisionByEventID(r.Context(), actor, chi.URLParam(r, "event_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", dec)
}

func (h *Handler) submitDispute(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.SubmitDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	d, err := h.service.SubmitDispute(r.Context(), actor, application.SubmitDisputeInput{DecisionID: strings.TrimSpace(req.DecisionID), SubmittedBy: strings.TrimSpace(req.SubmittedBy), EvidenceURL: strings.TrimSpace(req.EvidenceURL)})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", d)
}

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	m, err := h.service.GetMetrics(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", m)
}
