package http

import (
	"encoding/json"
	"net/http"

	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/contracts"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) scanLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
		return
	}
	var req contracts.ScanLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.ScanLicense(r.Context(), actorFromContext(r.Context()), application.ScanLicenseInput{
		SubmissionID:      req.SubmissionID,
		CreatorID:         req.CreatorID,
		MediaType:         req.MediaType,
		MediaURL:          req.MediaURL,
		DeclaredLicenseID: req.DeclaredLicenseID,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "license scan completed", out)
}

func (h *Handler) fileAppeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
		return
	}
	var req contracts.FileAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.FileAppeal(r.Context(), actorFromContext(r.Context()), application.FileAppealInput{
		SubmissionID:       req.SubmissionID,
		HoldID:             req.HoldID,
		CreatorID:          req.CreatorID,
		CreatorExplanation: req.CreatorExplanation,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "license appeal filed", out)
}

func (h *Handler) receiveDMCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", requestIDFromContext(r.Context()))
		return
	}
	var req contracts.DMCATakedownRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	out, err := h.service.ReceiveDMCATakedown(r.Context(), actorFromContext(r.Context()), application.DMCATakedownInput{
		SubmissionID:     req.SubmissionID,
		RightsHolderName: req.RightsHolderName,
		ContactEmail:     req.ContactEmail,
		Reference:        req.Reference,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "dmca takedown accepted", out)
}
