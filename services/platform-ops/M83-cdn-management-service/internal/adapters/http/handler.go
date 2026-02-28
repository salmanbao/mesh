package http

import (
	"encoding/json"
	"net/http"

	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createConfig(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	config, err := h.service.CreateConfig(r.Context(), actor, application.CreateConfigInput{
		Provider:     req.Provider,
		Config:       req.Config,
		HeaderRules:  req.HeaderRules,
		SignedURLTTL: req.SignedURLTTL,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", config)
}

func (h *Handler) listConfigs(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	items, err := h.service.ListConfigs(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", items)
}

func (h *Handler) purge(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.PurgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	purge, err := h.service.Purge(r.Context(), actor, application.PurgeInput{Scope: req.Scope, Target: req.Target})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", purge)
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Metrics(r.Context())
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", data)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, http.StatusOK, "ok", map[string]string{"status": "ok"})
}
