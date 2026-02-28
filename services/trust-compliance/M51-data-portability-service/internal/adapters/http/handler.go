package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createExport(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateExport(r.Context(), actorFromContext(r.Context()), application.CreateExportInput{
		UserID: req.UserID,
		Format: req.Format,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "export request created", toResponse(row))
}

func (h *Handler) createErase(w http.ResponseWriter, r *http.Request) {
	var req contracts.EraseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateEraseRequest(r.Context(), actorFromContext(r.Context()), application.EraseInput{
		UserID: req.UserID,
		Reason: req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "erasure request created", toResponse(row))
}

func (h *Handler) getExport(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.PathValue("request_id"))
	row, err := h.service.GetExport(r.Context(), actorFromContext(r.Context()), requestID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "export request", toResponse(row))
}

func (h *Handler) listExports(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	rows, err := h.service.ListExports(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.URL.Query().Get("user_id")), limit)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.ExportHistoryResponse{Items: make([]contracts.ExportRequestResponse, 0, len(rows))}
	for _, row := range rows {
		resp.Items = append(resp.Items, toResponse(row))
	}
	writeSuccess(w, http.StatusOK, "export history", resp)
}

func toResponse(row domain.ExportRequest) contracts.ExportRequestResponse {
	out := contracts.ExportRequestResponse{
		RequestID:    row.RequestID,
		UserID:       row.UserID,
		RequestType:  row.RequestType,
		Format:       row.Format,
		Status:       row.Status,
		Reason:       row.Reason,
		RequestedAt:  row.RequestedAt.UTC().Format(time.RFC3339),
		DownloadURL:  row.DownloadURL,
		FailureCause: row.FailureCause,
	}
	if row.CompletedAt != nil {
		out.CompletedAt = row.CompletedAt.UTC().Format(time.RFC3339)
	}
	return out
}
