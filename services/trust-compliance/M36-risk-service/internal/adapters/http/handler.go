package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/contracts"
)

func (h *Handler) getSellerRiskDashboard(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	dashboard, err := h.service.GetSellerRiskDashboard(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", dashboard)
}

func (h *Handler) fileDispute(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.FileDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.FileDispute(r.Context(), actor, application.FileDisputeInput{
		TransactionID: strings.TrimSpace(req.TransactionID),
		DisputeType:   strings.TrimSpace(req.DisputeType),
		Reason:        strings.TrimSpace(req.Reason),
		BuyerClaim:    strings.TrimSpace(req.BuyerClaim),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", row)
}

func (h *Handler) submitDisputeEvidence(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.SubmitDisputeEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.SubmitDisputeEvidence(r.Context(), actor, chi.URLParam(r, "dispute_id"), application.SubmitEvidenceInput{
		Filename:    strings.TrimSpace(req.Filename),
		Description: strings.TrimSpace(req.Description),
		FileURL:     strings.TrimSpace(req.FileURL),
		SizeBytes:   req.SizeBytes,
		MimeType:    strings.TrimSpace(req.MimeType),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", row)
}

func (h *Handler) handleChargebackWebhook(w http.ResponseWriter, r *http.Request) {
	var req contracts.ChargebackWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	bearer := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(bearer), "bearer ") {
		bearer = strings.TrimSpace(bearer[7:])
	}
	result, err := h.service.HandleChargebackWebhook(r.Context(), bearer, application.ChargebackInput{
		EventID:          strings.TrimSpace(req.EventID),
		EventType:        strings.TrimSpace(req.EventType),
		OccurredAt:       strings.TrimSpace(req.OccurredAt),
		SourceService:    strings.TrimSpace(req.SourceService),
		TraceID:          strings.TrimSpace(req.TraceID),
		SchemaVersion:    strings.TrimSpace(req.SchemaVersion),
		PartitionKeyPath: strings.TrimSpace(req.PartitionKeyPath),
		PartitionKey:     strings.TrimSpace(req.PartitionKey),
		Amount:           req.Data.Amount,
		ChargeID:         strings.TrimSpace(req.Data.ChargeID),
		Currency:         strings.TrimSpace(req.Data.Currency),
		DisputeReason:    strings.TrimSpace(req.Data.DisputeReason),
		SellerID:         strings.TrimSpace(req.Data.SellerID),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusAccepted, "", result)
}
