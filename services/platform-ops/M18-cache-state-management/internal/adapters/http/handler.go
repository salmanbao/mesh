package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) getCache(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	item, err := h.service.GetCache(r.Context(), actor, chi.URLParam(r, "key"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.GetCacheResponse{Key: item.Key, Found: item.Found, TTLSeconds: item.TTLSeconds}
	if item.Found {
		resp.Value = append([]byte(nil), item.Value...)
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) putCache(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.PutCacheRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	item, err := h.service.PutCache(r.Context(), actor, chi.URLParam(r, "key"), req.Value, req.TTLSeconds)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.PutCacheResponse{Key: item.Key, StoredAt: time.Now().UTC().Format(time.RFC3339), TTLSeconds: item.TTLSeconds})
}

func (h *Handler) deleteCache(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	deleted, err := h.service.DeleteCache(r.Context(), actor, chi.URLParam(r, "key"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.DeleteCacheResponse{Key: chi.URLParam(r, "key"), Deleted: deleted})
}

func (h *Handler) invalidateCache(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.InvalidateCacheRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	count, err := h.service.InvalidateCache(r.Context(), actor, req.Keys)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.InvalidateCacheResponse{InvalidatedCount: count})
}

func (h *Handler) getMetrics(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	m, err := h.service.GetMetrics(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.MetricsResponse{Hits: m.Hits, Misses: m.Misses, Evictions: m.Evictions, MemoryUsedBytes: m.MemoryUsedBytes})
}

func (h *Handler) getHealth(w http.ResponseWriter, r *http.Request) {
	health, err := h.service.GetHealth(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	status := http.StatusOK
	if health.Status != "healthy" {
		status = http.StatusServiceUnavailable
	}
	writeSuccess(w, status, "", health)
}
