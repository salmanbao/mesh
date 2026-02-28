package http

import (
	"encoding/json"
	"net/http"

	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createPlan(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	plan, err := h.service.CreatePlan(r.Context(), actor, application.CreatePlanInput{ServiceName: req.ServiceName, Environment: req.Environment, Version: req.Version, Plan: req.Plan, DryRun: req.DryRun, RiskLevel: req.RiskLevel})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", plan)
}

func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	plans, err := h.service.ListPlans(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", plans)
}

func (h *Handler) createRun(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	run, err := h.service.CreateRun(r.Context(), actor, application.CreateRunInput{PlanID: req.PlanID})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", run)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Health(r.Context())
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "ok", data)
}
