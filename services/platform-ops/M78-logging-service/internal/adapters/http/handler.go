package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) ingestLogs(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var req contracts.IngestLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}

	defaultInstanceID := strings.TrimSpace(r.Header.Get("X-Instance-Id"))
	defaultTraceID := strings.TrimSpace(r.Header.Get("X-Trace-Id"))

	logs := make([]application.IngestLogRecordInput, 0, len(req.Logs))
	for _, item := range req.Logs {
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_input", "invalid timestamp")
			return
		}
		instanceID := strings.TrimSpace(item.InstanceID)
		if instanceID == "" {
			instanceID = defaultInstanceID
		}
		traceID := strings.TrimSpace(item.TraceID)
		if traceID == "" {
			traceID = defaultTraceID
		}
		logs = append(logs, application.IngestLogRecordInput{
			Timestamp:  ts,
			Level:      item.Level,
			Service:    item.Service,
			InstanceID: instanceID,
			TraceID:    traceID,
			Message:    item.Message,
			UserID:     item.UserID,
			ErrorCode:  item.ErrorCode,
			Tags:       item.Tags,
		})
	}

	out, err := h.service.IngestLogs(r.Context(), actor, application.IngestLogsInput{Logs: logs})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}

	writeSuccess(w, http.StatusAccepted, "", contracts.IngestLogsResponse{Ingested: out.Ingested})
}

func (h *Handler) searchLogs(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	from, err := parseOptionalTime(r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid from timestamp")
		return
	}
	to, err := parseOptionalTime(r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid to timestamp")
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)

	rows, err := h.service.SearchLogs(r.Context(), actor, application.SearchLogsInput{
		Service: strings.TrimSpace(r.URL.Query().Get("service")),
		Level:   strings.TrimSpace(r.URL.Query().Get("level")),
		From:    from,
		To:      to,
		Q:       strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:   limit,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}

	items := make([]contracts.SearchLogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.SearchLogItem{
			EventID:    row.EventID,
			Timestamp:  row.Timestamp.Format(time.RFC3339),
			Level:      row.Level,
			Service:    row.Service,
			InstanceID: row.InstanceID,
			TraceID:    row.TraceID,
			Message:    row.Message,
			UserID:     row.UserID,
			ErrorCode:  row.ErrorCode,
			Tags:       row.Tags,
			Redacted:   row.Redacted,
			IngestedAt: row.IngestedAt.Format(time.RFC3339),
		})
	}
	writeSuccess(w, http.StatusOK, "", contracts.SearchLogsResponse{Items: items})
}

func (h *Handler) createExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var req contracts.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	out, err := h.service.CreateExport(r.Context(), actor, application.CreateExportInput{
		Query:  req.Query,
		Format: req.Format,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateExportResponse{ExportID: out.ExportID, Status: out.Status})
}

func (h *Handler) getExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	row, err := h.service.GetExport(r.Context(), actor, chi.URLParam(r, "export_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	var query map[string]any
	if len(row.Query) > 0 {
		_ = json.Unmarshal(row.Query, &query)
	}
	resp := contracts.ExportDetailResponse{
		ExportID:    row.ExportID,
		RequestedBy: row.RequestedBy,
		Query:       query,
		Format:      row.Format,
		Status:      row.Status,
		FileURL:     row.FileURL,
		CreatedAt:   row.CreatedAt.Format(time.RFC3339),
	}
	if row.CompletedAt != nil {
		resp.CompletedAt = row.CompletedAt.UTC().Format(time.RFC3339)
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) createAlertRule(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var req contracts.CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreateAlertRule(r.Context(), actor, application.CreateAlertRuleInput{
		Service:   req.Service,
		Condition: req.Condition,
		Severity:  req.Severity,
		Enabled:   req.Enabled,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", toAlertRuleResponse(row))
}

func (h *Handler) listAlertRules(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	onlyEnabled := true
	if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
		onlyEnabled = strings.EqualFold(raw, "true") || raw == "1"
	}
	rows, err := h.service.ListAlertRules(r.Context(), actor, onlyEnabled)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	items := make([]contracts.AlertRuleResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAlertRuleResponse(row))
	}
	writeSuccess(w, http.StatusOK, "", contracts.ListAlertRulesResponse{Items: items})
}

func (h *Handler) queryAudit(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	rows, err := h.service.QueryAudit(r.Context(), actor, application.AuditQueryInput{
		ActorID:    strings.TrimSpace(r.URL.Query().Get("actor_id")),
		ActionType: strings.TrimSpace(r.URL.Query().Get("action_type")),
		Limit:      parseIntDefault(r.URL.Query().Get("limit"), 100),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	out := make([]contracts.AuditLogItem, 0, len(rows.Logs))
	for _, row := range rows.Logs {
		out = append(out, contracts.AuditLogItem{
			AuditID:    row.AuditID,
			ActorID:    row.ActorID,
			ActionType: row.ActionType,
			ActionAt:   row.ActionAt.Format(time.RFC3339),
			IPAddress:  row.IPAddress,
			Details:    row.Details,
		})
	}
	writeSuccess(w, http.StatusOK, "", contracts.AuditQueryResponse{Logs: out})
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
	writeSuccess(w, http.StatusOK, "", contracts.ServiceMetricsResponse{
		Hits:            m.Hits,
		Misses:          m.Misses,
		Evictions:       m.Evictions,
		MemoryUsedBytes: m.MemoryUsedBytes,
	})
}

func actorFromRequest(r *http.Request) application.Actor {
	actor := actorFromContext(r.Context())
	actor.IPAddress = remoteIP(r.RemoteAddr)
	actor.UserAgent = r.UserAgent()
	return actor
}

func remoteIP(addr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return strings.TrimSpace(addr)
	}
	return host
}

func parseOptionalTime(raw string) (*time.Time, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	ts = ts.UTC()
	return &ts, nil
}

func parseIntDefault(raw string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		return v
	}
	return def
}

func toAlertRuleResponse(row domain.AlertRule) contracts.AlertRuleResponse {
	var condition map[string]any
	if len(row.Condition) > 0 {
		_ = json.Unmarshal(row.Condition, &condition)
	}
	return contracts.AlertRuleResponse{
		RuleID:    row.RuleID,
		Service:   row.Service,
		Condition: condition,
		Severity:  row.Severity,
		Enabled:   row.Enabled,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339),
	}
}
