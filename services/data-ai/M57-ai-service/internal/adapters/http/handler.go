package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) analyze(w http.ResponseWriter, r *http.Request) {
	var req contracts.AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.Analyze(r.Context(), actorFromContext(r.Context()), application.AnalyzeInput{
		UserID:       req.UserID,
		ContentID:    req.ContentID,
		Content:      req.Content,
		ModelID:      req.ModelID,
		ModelVersion: req.ModelVersion,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "analysis completed", toPredictionResponse(row))
}

func (h *Handler) batchAnalyze(w http.ResponseWriter, r *http.Request) {
	var req contracts.BatchAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	items := make([]application.BatchItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, application.BatchItemInput{
			ContentID: item.ContentID,
			Content:   item.Content,
		})
	}
	job, err := h.service.BatchAnalyze(r.Context(), actorFromContext(r.Context()), application.BatchAnalyzeInput{
		UserID:       req.UserID,
		ModelID:      req.ModelID,
		ModelVersion: req.ModelVersion,
		Items:        items,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "batch analysis completed", toBatchResponse(job))
}

func (h *Handler) getBatchStatus(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("job_id"))
	job, err := h.service.GetBatchStatus(r.Context(), actorFromContext(r.Context()), jobID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "batch status", toBatchResponse(job))
}

func toPredictionResponse(row domain.Prediction) contracts.PredictionResponse {
	return contracts.PredictionResponse{
		PredictionID: row.PredictionID,
		UserID:       row.UserID,
		ContentID:    row.ContentID,
		Label:        row.Label,
		Confidence:   row.Confidence,
		Flagged:      row.Flagged,
		ModelID:      row.ModelID,
		ModelVersion: row.ModelVersion,
		CreatedAt:    row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toBatchResponse(job domain.BatchJob) contracts.BatchStatusResponse {
	out := contracts.BatchStatusResponse{
		JobID:          job.JobID,
		UserID:         job.UserID,
		Status:         job.Status,
		ModelID:        job.ModelID,
		ModelVersion:   job.ModelVersion,
		RequestedCount: job.RequestedCount,
		CompletedCount: job.CompletedCount,
		CreatedAt:      job.CreatedAt.UTC().Format(time.RFC3339),
		StatusURL:      job.StatusURL,
		Predictions:    make([]contracts.PredictionResponse, 0, len(job.Predictions)),
	}
	if job.CompletedAt != nil {
		out.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339)
	}
	for _, row := range job.Predictions {
		out.Predictions = append(out.Predictions, toPredictionResponse(row))
	}
	return out
}
