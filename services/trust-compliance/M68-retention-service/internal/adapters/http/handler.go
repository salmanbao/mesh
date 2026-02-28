package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) listPolicies(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.ListPolicies(r.Context(), actorFromContext(r.Context()))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.PoliciesResponse{Items: make([]contracts.RetentionPolicyResponse, 0, len(rows))}
	for _, row := range rows {
		resp.Items = append(resp.Items, toPolicyResponse(row))
	}
	writeSuccess(w, http.StatusOK, "retention policies", resp)
}

func (h *Handler) createPolicy(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreatePolicy(r.Context(), actorFromContext(r.Context()), application.CreatePolicyInput{
		DataType:            req.DataType,
		RetentionYears:      req.RetentionYears,
		SoftDeleteGraceDays: req.SoftDeleteGraceDays,
		SelectiveRules:      req.SelectiveRules,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "retention policy created", toPolicyResponse(row))
}

func (h *Handler) createPreview(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreatePreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreatePreview(r.Context(), actorFromContext(r.Context()), application.CreatePreviewInput{
		PolicyID: req.PolicyID,
		DataType: req.DataType,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "deletion preview created", toPreviewResponse(row))
}

func (h *Handler) approvePreview(w http.ResponseWriter, r *http.Request) {
	var req contracts.ApprovePreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ApprovePreview(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("preview_id")), req.Reason)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "deletion preview approved", toScheduledDeletionResponse(row))
}

func (h *Handler) createLegalHold(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateLegalHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	var expiresAt *time.Time
	if raw := strings.TrimSpace(req.ExpiresAt); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid expires_at", requestIDFromContext(r.Context()))
			return
		}
		utc := parsed.UTC()
		expiresAt = &utc
	}
	row, err := h.service.CreateLegalHold(r.Context(), actorFromContext(r.Context()), application.CreateLegalHoldInput{
		EntityID:  req.EntityID,
		DataType:  req.DataType,
		Reason:    req.Reason,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "legal hold created", toLegalHoldResponse(row))
}

func (h *Handler) listLegalHolds(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.ListLegalHolds(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.LegalHoldsResponse{Items: make([]contracts.LegalHoldResponse, 0, len(rows))}
	for _, row := range rows {
		resp.Items = append(resp.Items, toLegalHoldResponse(row))
	}
	writeSuccess(w, http.StatusOK, "legal holds", resp)
}

func (h *Handler) createRestoration(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateRestorationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateRestoration(r.Context(), actorFromContext(r.Context()), application.CreateRestorationInput{
		EntityID:        req.EntityID,
		DataType:        req.DataType,
		Reason:          req.Reason,
		ArchiveLocation: req.ArchiveLocation,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "restoration request created", toRestorationResponse(row))
}

func (h *Handler) approveRestoration(w http.ResponseWriter, r *http.Request) {
	var req contracts.ApproveRestorationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ApproveRestoration(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("restoration_id")), req.Reason)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "restoration approved", toRestorationResponse(row))
}

func (h *Handler) complianceReport(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.ComplianceReport(r.Context(), actorFromContext(r.Context()))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "compliance report", contracts.ComplianceReportResponse{
		PolicyCount:           stats["policy_count"],
		ActiveLegalHolds:      stats["active_legal_holds"],
		PendingDeletions:      stats["pending_deletions"],
		TotalScheduledRecords: stats["total_scheduled_records"],
		RestorationRequests:   stats["restoration_requests"],
	})
}

func toPolicyResponse(row domain.RetentionPolicy) contracts.RetentionPolicyResponse {
	return contracts.RetentionPolicyResponse{
		PolicyID:            row.PolicyID,
		DataType:            row.DataType,
		RetentionYears:      row.RetentionYears,
		SoftDeleteGraceDays: row.SoftDeleteGraceDays,
		SelectiveRules:      row.SelectiveRules,
		Status:              row.Status,
		CreatedBy:           row.CreatedBy,
		CreatedAt:           row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toPreviewResponse(row domain.DeletionPreview) contracts.DeletionPreviewResponse {
	out := contracts.DeletionPreviewResponse{
		PreviewID:            row.PreviewID,
		PolicyID:             row.PolicyID,
		DataType:             row.DataType,
		TotalRecordsToDelete: row.TotalRecordsToDelete,
		EstimatedBytes:       row.EstimatedBytes,
		WillBeArchivedTo:     row.WillBeArchivedTo,
		Status:               row.Status,
		RequestedBy:          row.RequestedBy,
		CreatedAt:            row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.ApprovedAt != nil {
		out.ApprovedAt = row.ApprovedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toScheduledDeletionResponse(row domain.ScheduledDeletion) contracts.ScheduledDeletionResponse {
	return contracts.ScheduledDeletionResponse{
		DeletionID:   row.DeletionID,
		PreviewID:    row.PreviewID,
		PolicyID:     row.PolicyID,
		DataType:     row.DataType,
		Status:       row.Status,
		RecordsCount: row.RecordsCount,
		Reason:       row.Reason,
		ScheduledAt:  row.ScheduledAt.UTC().Format(time.RFC3339),
	}
}

func toLegalHoldResponse(row domain.LegalHold) contracts.LegalHoldResponse {
	out := contracts.LegalHoldResponse{
		HoldID:    row.HoldID,
		EntityID:  row.EntityID,
		DataType:  row.DataType,
		Reason:    row.Reason,
		Status:    row.Status,
		IssuedBy:  row.IssuedBy,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.ExpiresAt != nil {
		out.ExpiresAt = row.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toRestorationResponse(row domain.RestorationRequest) contracts.RestorationResponse {
	out := contracts.RestorationResponse{
		RestorationID:   row.RestorationID,
		EntityID:        row.EntityID,
		DataType:        row.DataType,
		Reason:          row.Reason,
		ArchiveLocation: row.ArchiveLocation,
		Status:          row.Status,
		RequestedBy:     row.RequestedBy,
		CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.ApprovedAt != nil {
		out.ApprovedAt = row.ApprovedAt.UTC().Format(time.RFC3339)
	}
	return out
}
