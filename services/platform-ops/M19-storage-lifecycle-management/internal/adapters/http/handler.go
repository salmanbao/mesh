package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createPolicy(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateStoragePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreatePolicy(r.Context(), actor, application.CreatePolicyInput{
		PolicyID:        req.PolicyID,
		Scope:           req.Scope,
		TierFrom:        req.TierFrom,
		TierTo:          req.TierTo,
		AfterDays:       req.AfterDays,
		LegalHoldExempt: req.LegalHoldExempt,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateStoragePolicyResponse{
		PolicyID:  row.PolicyID,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) analyticsSummary(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	summary, err := h.service.GetAnalyticsSummary(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	resp := contracts.AnalyticsSummaryResponse{
		TotalObjects: summary.TotalObjects,
		ByTier:       summary.ByTier,
		MonthlyCost:  summary.MonthlyCost,
	}
	if !summary.LastRunAt.IsZero() {
		resp.LastRunAt = summary.LastRunAt.Format(time.RFC3339)
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) moveToGlacier(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.MoveToGlacierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	job, err := h.service.MoveToGlacier(r.Context(), actor, application.MoveToGlacierInput{
		FileID:            req.FileID,
		SubmissionID:      req.SubmissionID,
		CampaignID:        req.CampaignID,
		SourceBucket:      req.SourceBucket,
		SourceKey:         req.SourceKey,
		DestinationBucket: req.DestinationBucket,
		DestinationKey:    req.DestinationKey,
		ChecksumMD5:       req.ChecksumMD5,
		FileSizeBytes:     req.FileSizeBytes,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusAccepted, "", contracts.MoveToGlacierResponse{
		JobID:   job.JobID,
		FileID:  job.FileID,
		Status:  job.Status,
		Message: job.Message,
	})
}

func (h *Handler) scheduleDeletion(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.ScheduleDeletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	batch, err := h.service.ScheduleDeletion(r.Context(), actor, application.ScheduleDeletionInput{
		CampaignID:       req.CampaignID,
		DeletionType:     req.DeletionType,
		DaysAfterClosure: req.DaysAfterClosure,
		FileIDs:          req.FileIDs,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusAccepted, "", contracts.ScheduleDeletionResponse{
		BatchID:      batch.BatchID,
		FileCount:    batch.FileCount,
		ScheduledFor: batch.ScheduledFor.Format(time.RFC3339),
		Status:       batch.Status,
	})
}

func (h *Handler) queryDeletionAudit(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var startPtr, endPtr *time.Time
	if raw := strings.TrimSpace(r.URL.Query().Get("start_date")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			tt := t.UTC()
			startPtr = &tt
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("end_date")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			tt := t.UTC()
			endPtr = &tt
		}
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	out, err := h.service.QueryDeletionAudit(r.Context(), actor, application.AuditQueryInput{
		FileID:     strings.TrimSpace(r.URL.Query().Get("file_id")),
		CampaignID: strings.TrimSpace(r.URL.Query().Get("campaign_id")),
		Action:     strings.TrimSpace(r.URL.Query().Get("action")),
		StartDate:  startPtr,
		EndDate:    endPtr,
		Limit:      limit,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	items := make([]contracts.DeletionAuditItem, 0, len(out.Records))
	for _, row := range out.Records {
		items = append(items, contracts.DeletionAuditItem{
			AuditID:       row.AuditID,
			FileID:        row.FileID,
			CampaignID:    row.CampaignID,
			Action:        row.Action,
			TriggeredBy:   row.TriggeredBy,
			FileSizeBytes: row.FileSizeBytes,
			Reason:        row.Reason,
			InitiatedAt:   row.InitiatedAt.Format(time.RFC3339),
			CompletedAt:   row.CompletedAt.Format(time.RFC3339),
		})
	}
	writeSuccess(w, http.StatusOK, "", contracts.DeletionAuditQueryResponse{
		Deletions:         items,
		TotalFilesDeleted: out.TotalFilesDeleted,
		TotalSizeFreed:    out.TotalSizeFreed,
	})
}

func (h *Handler) getHealth(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetHealth(r.Context())
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	status := http.StatusOK
	if out.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}
	writeSuccess(w, status, "", out)
}

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/plain") {
		payload, err := h.service.RenderPrometheusMetrics(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
		return
	}
	m, err := h.service.GetCacheMetrics(r.Context())
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.CacheMetricsResponse{
		Hits:            m.Hits,
		Misses:          m.Misses,
		Evictions:       m.Evictions,
		MemoryUsedBytes: m.MemoryUsedBytes,
	})
}

func _domainRef(_ domain.HealthReport) {}
