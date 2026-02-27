package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) ingestOTLP(w http.ResponseWriter, r *http.Request)   { h.ingest(w, r, "otlp") }
func (h *Handler) ingestZipkin(w http.ResponseWriter, r *http.Request) { h.ingest(w, r, "zipkin") }

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request, format string) {
	actor := actorFromContext(r.Context())
	var req contracts.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	spans := make([]domain.IngestedSpan, 0, len(req.Spans))
	for _, row := range req.Spans {
		start, _ := time.Parse(time.RFC3339, strings.TrimSpace(row.StartTime))
		end, _ := time.Parse(time.RFC3339, strings.TrimSpace(row.EndTime))
		spans = append(spans, domain.IngestedSpan{TraceID: row.TraceID, SpanID: row.SpanID, ParentSpanID: row.ParentSpanID, ServiceName: row.ServiceName, OperationName: row.OperationName, StartTime: start, EndTime: end, Error: row.Error, HTTPStatusCode: row.HTTPStatusCode, Tags: row.Tags, Environment: row.Environment})
	}
	_, err := h.service.IngestSpans(r.Context(), actor, application.IngestInput{Format: format, Spans: spans})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	if format == "otlp" {
		writeAccepted(w, "spans queued")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) searchTraces(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	q := application.SearchInput{TraceID: strings.TrimSpace(r.URL.Query().Get("trace_id")), ServiceName: strings.TrimSpace(r.URL.Query().Get("service_name")), Limit: 50}
	if raw := strings.TrimSpace(r.URL.Query().Get("error")); raw != "" {
		v := strings.EqualFold(raw, "true") || raw == "1"
		q.ErrorOnly = &v
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("duration_gt_ms")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			q.DurationGTMS = &v
		}
	}
	rows, err := h.service.SearchTraces(r.Context(), actor, q)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	out := make([]contracts.TraceSearchItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, contracts.TraceSearchItem{TraceID: row.TraceID, DurationMS: row.DurationMS, Error: row.Error})
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) getTrace(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	detail, err := h.service.GetTraceDetail(r.Context(), actor, chi.URLParam(r, "trace_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	tagMap := map[string]map[string]string{}
	for _, t := range detail.Tags {
		if _, ok := tagMap[t.SpanID]; !ok {
			tagMap[t.SpanID] = map[string]string{}
		}
		tagMap[t.SpanID][t.Key] = t.Value
	}
	spans := make([]contracts.SpanDetailResponse, 0, len(detail.Spans))
	for _, sp := range detail.Spans {
		spans = append(spans, contracts.SpanDetailResponse{SpanID: sp.SpanID, ServiceName: sp.ServiceName, Operation: sp.OperationName, ParentSpanID: sp.ParentSpanID, DurationMS: sp.DurationMS, Error: sp.Error || sp.HTTPStatusCode >= 500, Tags: tagMap[sp.SpanID]})
	}
	writeSuccess(w, http.StatusOK, "", contracts.TraceDetailResponse{TraceID: detail.Trace.TraceID, Spans: spans})
}

func (h *Handler) createSamplingPolicy(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateSamplingPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreateSamplingPolicy(r.Context(), actor, application.CreateSamplingPolicyInput{ServiceName: req.ServiceName, RuleType: req.RuleType, Probability: req.Probability, MaxTracesPerMin: req.MaxTracesPerMin})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateSamplingPolicyResponse{PolicyID: row.PolicyID})
}

func (h *Handler) listSamplingPolicies(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	rows, err := h.service.ListSamplingPolicies(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	out := make([]contracts.SamplingPolicyResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, contracts.SamplingPolicyResponse{PolicyID: p.PolicyID, ServiceName: p.ServiceName, RuleType: p.RuleType, Probability: p.Probability, MaxTracesPerMin: p.MaxTracesPerMin, Enabled: p.Enabled})
	}
	writeSuccess(w, http.StatusOK, "", out)
}

func (h *Handler) createExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	job, err := h.service.CreateExport(r.Context(), actor, application.CreateExportInput{TraceID: req.TraceID, Format: req.Format, Filters: req.Filters})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.ExportResponse{ExportID: job.ExportID, Status: job.Status, OutputURI: job.OutputURI, ErrorMessage: job.ErrorMessage})
}

func (h *Handler) getExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	job, err := h.service.GetExport(r.Context(), actor, chi.URLParam(r, "export_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.ExportResponse{ExportID: job.ExportID, Status: job.Status, OutputURI: job.OutputURI, ErrorMessage: job.ErrorMessage})
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
	writeSuccess(w, http.StatusOK, "", contracts.CacheMetricsResponse{Hits: m.Hits, Misses: m.Misses, Evictions: m.Evictions, MemoryUsedBytes: m.MemoryUsedBytes})
}
