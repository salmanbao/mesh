package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
)

func (h *Handler) createDispute(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	dispute, err := h.service.CreateDispute(r.Context(), actor, application.CreateDisputeInput{
		DisputeType:       strings.TrimSpace(req.DisputeType),
		TransactionID:     strings.TrimSpace(req.TransactionID),
		ReasonCategory:    strings.TrimSpace(req.ReasonCategory),
		JustificationText: strings.TrimSpace(req.JustificationText),
		RequestedAmount:   req.RequestedAmount,
		EvidenceFiles:     mapEvidenceFiles(req.EvidenceFiles),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	hoursRemaining := dispute.SLAHoursTarget
	if dispute.ExpectedResolution != nil {
		hoursRemaining = maxInt(0, int(dispute.ExpectedResolution.Sub(time.Now().UTC()).Hours()))
	}
	writeSuccess(w, http.StatusCreated, "", map[string]any{
		"dispute_id":          dispute.DisputeID,
		"status":              dispute.Status,
		"assigned_agent":      dispute.AssignedAgentID,
		"expected_resolution": dispute.ExpectedResolution,
		"sla_hours_remaining": hoursRemaining,
	})
}

func (h *Handler) getDispute(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	detail, err := h.service.GetDispute(r.Context(), actor, chi.URLParam(r, "dispute_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	hoursRemaining := 0
	if detail.Dispute.ExpectedResolution != nil {
		hoursRemaining = maxInt(0, int(detail.Dispute.ExpectedResolution.Sub(time.Now().UTC()).Hours()))
	}
	writeSuccess(w, http.StatusOK, "", map[string]any{
		"dispute_id":         detail.Dispute.DisputeID,
		"dispute_type":       detail.Dispute.DisputeType,
		"status":             detail.Dispute.Status,
		"priority":           detail.Dispute.Priority,
		"transaction_id":     detail.Dispute.TransactionID,
		"requested_amount":   detail.Dispute.RequestedAmount,
		"justification_text": detail.Dispute.JustificationText,
		"assigned_agent":     map[string]any{"name": detail.Dispute.AssignedAgentID},
		"sla":                map[string]any{"target_hours": detail.Dispute.SLAHoursTarget, "hours_remaining": hoursRemaining, "breached": detail.Dispute.SLABreached},
		"created_at":         detail.Dispute.CreatedAt,
		"updated_at":         detail.Dispute.UpdatedAt,
	})
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	msg, err := h.service.SendMessage(r.Context(), actor, chi.URLParam(r, "dispute_id"), application.SendMessageInput{MessageBody: strings.TrimSpace(req.MessageBody), Attachments: mapEvidenceFiles(req.Attachments)})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", map[string]any{"message_id": msg.MessageID, "created_at": msg.CreatedAt})
}

func (h *Handler) approveDispute(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ApproveDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	dispute, err := h.service.ApproveDispute(r.Context(), actor, chi.URLParam(r, "dispute_id"), application.ApproveDisputeInput{RefundAmount: req.RefundAmount, ApprovalReason: strings.TrimSpace(req.ApprovalReason), ResolutionNotes: strings.TrimSpace(req.ResolutionNotes)})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]any{"dispute_id": dispute.DisputeID, "status": dispute.Status, "resolution_type": dispute.ResolutionType, "refund_amount": dispute.ApprovedRefundAmount, "processed_at": dispute.ResolvedAt})
}

func mapEvidenceFiles(in []contracts.EvidenceFile) []domain.EvidenceFile {
	out := make([]domain.EvidenceFile, 0, len(in))
	for _, item := range in {
		out = append(out, domain.EvidenceFile{Filename: strings.TrimSpace(item.Filename), FileURL: strings.TrimSpace(item.FileURL)})
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
