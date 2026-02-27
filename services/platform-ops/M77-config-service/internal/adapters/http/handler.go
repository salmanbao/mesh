package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	out, err := h.service.GetConfig(r.Context(), actor, application.GetConfigInput{
		Environment:  strings.TrimSpace(r.URL.Query().Get("env")),
		ServiceScope: strings.TrimSpace(r.URL.Query().Get("service")),
		UserID:       strings.TrimSpace(r.URL.Query().Get("user_id")),
		Role:         strings.TrimSpace(r.URL.Query().Get("role")),
		Tier:         strings.TrimSpace(r.URL.Query().Get("tier")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.GetConfigResponse{
		Environment:  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("env"))),
		ServiceScope: domain.NormalizeServiceScope(r.URL.Query().Get("service")),
		Values:       out,
	})
}

func (h *Handler) patchConfig(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	var req contracts.PatchConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	out, err := h.service.PatchConfig(r.Context(), actor, application.PatchConfigInput{
		Key:          chi.URLParam(r, "key"),
		Environment:  req.Environment,
		ServiceScope: req.ServiceScope,
		ValueType:    req.ValueType,
		Value:        req.Value,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.PatchConfigResponse{
		Key:          out.Key,
		Environment:  out.Environment,
		ServiceScope: out.ServiceScope,
		Version:      out.Version,
	})
}

func (h *Handler) importConfig(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	var req contracts.ImportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	entries := make([]application.ImportConfigEntry, 0, len(req.Entries))
	for _, item := range req.Entries {
		entries = append(entries, application.ImportConfigEntry{
			Key:       item.Key,
			ValueType: item.ValueType,
			Value:     item.Value,
		})
	}
	applied, err := h.service.ImportConfig(r.Context(), actor, application.ImportConfigInput{
		Environment:  req.Environment,
		ServiceScope: req.ServiceScope,
		Entries:      entries,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.ImportConfigResponse{AppliedCount: applied})
}

func (h *Handler) exportConfig(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	out, err := h.service.ExportConfig(r.Context(), actor, application.ExportConfigInput{
		Environment:  strings.TrimSpace(r.URL.Query().Get("env")),
		ServiceScope: strings.TrimSpace(r.URL.Query().Get("service")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	meta := make(map[string]contracts.ExportMeta, len(out.Meta))
	for k, v := range out.Meta {
		meta[k] = contracts.ExportMeta{
			ValueType:  v.ValueType,
			UpdatedAt:  v.UpdatedAt.Format(time.RFC3339),
			KeyVersion: v.KeyVersion,
		}
	}
	writeSuccess(w, http.StatusOK, "", contracts.ExportConfigResponse{
		Version:      out.Version,
		GeneratedAt:  out.GeneratedAt.Format(time.RFC3339),
		Environment:  out.Environment,
		ServiceScope: out.ServiceScope,
		Values:       out.Values,
		Meta:         meta,
	})
}

func (h *Handler) rollbackConfig(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	var req contracts.RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	out, err := h.service.RollbackConfig(r.Context(), actor, application.RollbackConfigInput{
		Key:          req.Key,
		Environment:  req.Environment,
		ServiceScope: req.ServiceScope,
		Version:      req.Version,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.RollbackResponse{
		Key:          out.Key,
		Environment:  out.Environment,
		ServiceScope: out.ServiceScope,
		Version:      out.Version,
		RolledBackTo: out.RolledBackTo,
	})
}

func (h *Handler) upsertRolloutRule(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	var req contracts.RolloutRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	out, err := h.service.CreateRolloutRule(r.Context(), actor, application.CreateRolloutRuleInput{
		Key:        req.Key,
		RuleType:   req.RuleType,
		Percentage: req.Percentage,
		Role:       req.Role,
		Tier:       req.Tier,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.RolloutRuleResponse{
		RuleID:    out.RuleID,
		Key:       out.KeyName,
		RuleType:  out.RuleType,
		CreatedAt: out.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) queryAudit(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	actor.IPAddress = r.RemoteAddr
	actor.UserAgent = r.UserAgent()

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	out, err := h.service.QueryAudit(r.Context(), actor, application.AuditQueryInput{
		KeyName:      strings.TrimSpace(r.URL.Query().Get("key")),
		Environment:  strings.TrimSpace(r.URL.Query().Get("env")),
		ServiceScope: strings.TrimSpace(r.URL.Query().Get("service")),
		ActorID:      strings.TrimSpace(r.URL.Query().Get("actor_id")),
		Limit:        limit,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error())
		return
	}
	items := make([]contracts.AuditLogItem, 0, len(out.Logs))
	for _, row := range out.Logs {
		items = append(items, contracts.AuditLogItem{
			AuditID:      row.AuditID,
			ActionType:   row.ActionType,
			KeyName:      row.KeyName,
			ActorID:      row.ActorID,
			Environment:  row.Environment,
			ServiceScope: row.ServiceScope,
			IPAddress:    row.IPAddress,
			UserAgent:    row.UserAgent,
			ActionAt:     row.ActionAt.Format(time.RFC3339),
		})
	}
	writeSuccess(w, http.StatusOK, "", contracts.AuditQueryResponse{Logs: items})
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

func _domainRef(_ domain.HealthReport) {}
