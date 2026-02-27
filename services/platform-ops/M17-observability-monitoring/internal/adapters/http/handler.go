package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) getHealth(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetHealth(r.Context())
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	checks := make(map[string]contracts.ComponentResponse, len(out.Checks))
	for name, row := range out.Checks {
		checks[name] = componentToResponse(row)
	}
	status := http.StatusOK
	if out.Status != domain.StatusHealthy {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, contracts.HealthCheckResponse{
		Status:        out.Status,
		Timestamp:     out.Timestamp.UTC().Format(time.RFC3339),
		UptimeSeconds: out.UptimeSeconds,
		Version:       out.Version,
		Checks:        checks,
	})
}

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.RenderPrometheusMetrics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}

func (h *Handler) listComponents(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.ListComponents(r.Context())
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	items := make([]contracts.NamedComponentResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, namedComponentToResponse(row))
	}
	writeSuccess(w, http.StatusOK, contracts.ComponentsListResponse{Items: items})
}

func (h *Handler) upsertComponent(w http.ResponseWriter, r *http.Request) {
	var req contracts.UpsertComponentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	out, err := h.service.UpsertComponent(r.Context(), actorFromContext(r.Context()), application.UpsertComponentInput{
		Name:             chi.URLParam(r, "name"),
		Status:           req.Status,
		Critical:         req.Critical,
		LatencyMS:        req.LatencyMS,
		BrokersConnected: req.BrokersConnected,
		Error:            req.Error,
		Metadata:         req.Metadata,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, contracts.UpsertComponentResponse{
		Name:        out.Name,
		Status:      out.Status,
		Critical:    out.Critical,
		LastChecked: out.LastChecked.UTC().Format(time.RFC3339),
	})
}

func componentToResponse(row domain.ComponentCheck) contracts.ComponentResponse {
	out := contracts.ComponentResponse{
		Status:           row.Status,
		LatencyMS:        row.LatencyMS,
		BrokersConnected: row.BrokersConnected,
		LastChecked:      row.LastChecked.UTC().Format(time.RFC3339),
		Error:            row.Error,
		Metadata:         row.Metadata,
	}
	return out
}

func namedComponentToResponse(row domain.ComponentCheck) contracts.NamedComponentResponse {
	return contracts.NamedComponentResponse{
		Name:             row.Name,
		Status:           row.Status,
		Critical:         row.Critical,
		LatencyMS:        row.LatencyMS,
		BrokersConnected: row.BrokersConnected,
		LastChecked:      row.LastChecked.UTC().Format(time.RFC3339),
		Error:            row.Error,
		Metadata:         row.Metadata,
	}
}
