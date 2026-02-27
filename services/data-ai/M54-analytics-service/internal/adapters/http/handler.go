package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/contracts"
)

func (h *Handler) getCreatorDashboard(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	data, err := h.service.GetCreatorDashboard(r.Context(), actor, application.DashboardInput{
		UserID:   strings.TrimSpace(r.URL.Query().Get("user_id")),
		DateFrom: strings.TrimSpace(r.URL.Query().Get("date_from")),
		DateTo:   strings.TrimSpace(r.URL.Query().Get("date_to")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", data)
}

func (h *Handler) getAdminFinancialReport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	data, err := h.service.GetAdminFinancialReport(r.Context(), actor, application.FinancialReportInput{
		DateFrom: strings.TrimSpace(r.URL.Query().Get("date_from")),
		DateTo:   strings.TrimSpace(r.URL.Query().Get("date_to")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", data)
}

func (h *Handler) requestExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	job, err := h.service.RequestExport(r.Context(), actor, application.ExportInput{
		ReportType: strings.TrimSpace(req.ReportType),
		Format:     strings.TrimSpace(req.Format),
		DateFrom:   strings.TrimSpace(req.DateFrom),
		DateTo:     strings.TrimSpace(req.DateTo),
		Filters:    req.Filters,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusAccepted, "export queued", map[string]interface{}{
		"export_id":    job.ExportID,
		"status":       job.Status,
		"download_url": job.DownloadURL,
		"created_at":   job.CreatedAt,
		"ready_at":     job.ReadyAt,
	})
}

func (h *Handler) getExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	exportID := chi.URLParam(r, "id")
	job, err := h.service.GetExport(r.Context(), actor, exportID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"export_id":    job.ExportID,
		"status":       job.Status,
		"download_url": job.DownloadURL,
		"created_at":   job.CreatedAt,
		"ready_at":     job.ReadyAt,
	})
}
