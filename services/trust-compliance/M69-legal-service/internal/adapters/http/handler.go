package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) uploadDocument(w http.ResponseWriter, r *http.Request) {
	var req contracts.UploadDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.UploadDocument(r.Context(), actorFromContext(r.Context()), application.UploadDocumentInput{
		DocumentType: req.DocumentType,
		FileName:     req.FileName,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "document uploaded", toDocumentResponse(row))
}

func (h *Handler) requestSignature(w http.ResponseWriter, r *http.Request) {
	var req contracts.RequestSignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.RequestSignature(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")), application.RequestSignatureInput{SignerUserID: req.SignerUserID})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "signature requested", toSignatureResponse(row))
}

func (h *Handler) createHold(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateHold(r.Context(), actorFromContext(r.Context()), application.CreateHoldInput{
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Reason:     req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "legal hold created", toHoldResponse(row))
}

func (h *Handler) checkHold(w http.ResponseWriter, r *http.Request) {
	entityType := strings.TrimSpace(r.URL.Query().Get("entity_type"))
	entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
	held, hold, err := h.service.CheckHold(r.Context(), actorFromContext(r.Context()), entityType, entityID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.HoldCheckResponse{EntityType: entityType, EntityID: entityID, Held: held}
	if hold != nil {
		resp.HoldID = hold.HoldID
	}
	writeSuccess(w, http.StatusOK, "hold check", resp)
}

func (h *Handler) releaseHold(w http.ResponseWriter, r *http.Request) {
	var req contracts.ReleaseHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.ReleaseHold(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")), req.Reason)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "legal hold released", toHoldResponse(row))
}

func (h *Handler) runComplianceScan(w http.ResponseWriter, r *http.Request) {
	var req contracts.ComplianceScanRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	row, err := h.service.RunComplianceScan(r.Context(), actorFromContext(r.Context()), application.ComplianceScanInput{ReportType: req.ReportType})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "compliance scan completed", toComplianceReportResponse(row))
}

func (h *Handler) getComplianceReport(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.GetComplianceReport(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("report_id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "compliance report", toComplianceReportResponse(row))
}

func (h *Handler) createDispute(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateDispute(r.Context(), actorFromContext(r.Context()), application.CreateDisputeInput{
		UserID:        req.UserID,
		OpposingParty: req.OpposingParty,
		DisputeReason: req.DisputeReason,
		AmountCents:   req.AmountCents,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "dispute created", toDisputeResponse(row))
}

func (h *Handler) getDispute(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.GetDispute(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("dispute_id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "dispute", toDisputeResponse(row))
}

func (h *Handler) createDMCANotice(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateDMCANoticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateDMCANotice(r.Context(), actorFromContext(r.Context()), application.CreateDMCANoticeInput{
		ContentID: req.ContentID,
		Claimant:  req.Claimant,
		Reason:    req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "dmca notice received", toDMCANoticeResponse(row))
}

func (h *Handler) generate1099(w http.ResponseWriter, r *http.Request) {
	var req contracts.GenerateFilingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.Generate1099(r.Context(), actorFromContext(r.Context()), application.GenerateFilingInput{
		UserID:  req.UserID,
		TaxYear: req.TaxYear,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "regulatory filing created", toFilingResponse(row))
}

func (h *Handler) getFilingStatus(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.GetFilingStatus(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("filing_id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "regulatory filing", toFilingResponse(row))
}

func toDocumentResponse(row domain.LegalDocument) contracts.LegalDocumentResponse {
	return contracts.LegalDocumentResponse{
		DocumentID:   row.DocumentID,
		DocumentType: row.DocumentType,
		FileName:     row.FileName,
		Status:       row.Status,
		UploadedBy:   row.UploadedBy,
		CreatedAt:    row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toSignatureResponse(row domain.SignatureRequest) contracts.SignatureResponse {
	return contracts.SignatureResponse{
		SignatureID:  row.SignatureID,
		DocumentID:   row.DocumentID,
		SignerUserID: row.SignerUserID,
		Status:       row.Status,
		RequestedBy:  row.RequestedBy,
		RequestedAt:  row.RequestedAt.UTC().Format(time.RFC3339),
	}
}

func toHoldResponse(row domain.LegalHold) contracts.HoldResponse {
	out := contracts.HoldResponse{
		HoldID:     row.HoldID,
		EntityType: row.EntityType,
		EntityID:   row.EntityID,
		Reason:     row.Reason,
		Status:     row.Status,
		IssuedBy:   row.IssuedBy,
		CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
	}
	if row.ReleasedAt != nil {
		out.ReleasedAt = row.ReleasedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func toComplianceReportResponse(row domain.ComplianceReport) contracts.ComplianceReportResponse {
	return contracts.ComplianceReportResponse{
		ReportID:      row.ReportID,
		ReportType:    row.ReportType,
		Status:        row.Status,
		FindingsCount: row.FindingsCount,
		DownloadURL:   row.DownloadURL,
		CreatedBy:     row.CreatedBy,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toDisputeResponse(row domain.Dispute) contracts.DisputeResponse {
	return contracts.DisputeResponse{
		DisputeID:     row.DisputeID,
		UserID:        row.UserID,
		OpposingParty: row.OpposingParty,
		DisputeReason: row.DisputeReason,
		AmountCents:   row.AmountCents,
		Status:        row.Status,
		EvidenceCount: row.EvidenceCount,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toDMCANoticeResponse(row domain.DMCANotice) contracts.DMCANoticeResponse {
	return contracts.DMCANoticeResponse{
		NoticeID:   row.NoticeID,
		ContentID:  row.ContentID,
		Claimant:   row.Claimant,
		Reason:     row.Reason,
		Status:     row.Status,
		ReceivedAt: row.ReceivedAt.UTC().Format(time.RFC3339),
	}
}

func toFilingResponse(row domain.RegulatoryFiling) contracts.FilingResponse {
	return contracts.FilingResponse{
		FilingID:      row.FilingID,
		FilingType:    row.FilingType,
		TaxYear:       row.TaxYear,
		UserID:        row.UserID,
		Status:        row.Status,
		TaxDocumentID: row.TaxDocumentID,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}
