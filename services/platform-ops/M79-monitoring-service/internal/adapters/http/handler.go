package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createAlertRule(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var req contracts.CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreateAlertRule(r.Context(), actor, application.CreateAlertRuleInput{
		Name:            req.Name,
		Query:           req.Query,
		Threshold:       req.Threshold,
		DurationSeconds: req.DurationSeconds,
		Severity:        req.Severity,
		Enabled:         req.Enabled,
		Service:         req.Service,
		Regex:           req.Regex,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateAlertRuleResponse{RuleID: row.RuleID})
}

func (h *Handler) listAlertRules(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	onlyEnabled := true
	if raw := strings.TrimSpace(r.URL.Query().Get("enabled")); raw != "" {
		onlyEnabled = strings.EqualFold(raw, "true") || raw == "1"
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)
	rows, err := h.service.ListAlertRules(r.Context(), actor, onlyEnabled, limit)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	items := make([]contracts.AlertRuleItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.AlertRuleItem{
			RuleID:          row.RuleID,
			Name:            row.Name,
			Query:           row.Query,
			Threshold:       row.Threshold,
			DurationSeconds: row.DurationSeconds,
			Severity:        row.Severity,
			Enabled:         row.Enabled,
			Service:         row.Service,
			Regex:           row.Regex,
			CreatedAt:       row.CreatedAt.Format(time.RFC3339),
		})
	}
	writeSuccess(w, http.StatusOK, "", contracts.ListAlertRulesResponse{Items: items})
}

func (h *Handler) listIncidents(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	rows, err := h.service.ListIncidents(r.Context(), actor, application.ListIncidentsInput{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:  parseIntDefault(r.URL.Query().Get("limit"), 100),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	items := make([]contracts.IncidentItem, 0, len(rows))
	for _, row := range rows {
		item := contracts.IncidentItem{
			IncidentID: row.IncidentID,
			AlertID:    row.AlertID,
			Service:    row.Service,
			Severity:   row.Severity,
			Status:     row.Status,
			Assignee:   row.Assignee,
			CreatedAt:  row.CreatedAt.Format(time.RFC3339),
		}
		if row.ResolvedAt != nil {
			item.ResolvedAt = row.ResolvedAt.UTC().Format(time.RFC3339)
		}
		items = append(items, item)
	}
	writeSuccess(w, http.StatusOK, "", contracts.ListIncidentsResponse{Items: items})
}

func (h *Handler) createSilence(w http.ResponseWriter, r *http.Request) {
	actor := actorFromRequest(r)
	var req contracts.CreateSilenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	startAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartAt))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid start_at")
		return
	}
	endAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndAt))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid end_at")
		return
	}
	row, err := h.service.CreateSilence(r.Context(), actor, application.CreateSilenceInput{
		RuleID:  req.RuleID,
		Reason:  req.Reason,
		StartAt: startAt,
		EndAt:   endAt,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateSilenceResponse{SilenceID: row.SilenceID})
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

func parseIntDefault(raw string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		return v
	}
	return def
}

func _domainRef(_ domain.HealthReport) {}
