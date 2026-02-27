package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/contracts"
)

func (h *Handler) getFunnel(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	resp, err := h.service.GetFunnel(r.Context(), actor, application.DateRangeInput{StartDate: strings.TrimSpace(r.URL.Query().Get("start_date")), EndDate: strings.TrimSpace(r.URL.Query().Get("end_date"))})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	resp, err := h.service.GetLeaderboard(r.Context(), actor, application.LeaderboardInput{Period: strings.TrimSpace(r.URL.Query().Get("period"))})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) getCohortRetention(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	resp, err := h.service.GetCohortRetention(r.Context(), actor, application.CohortInput{CohortStart: strings.TrimSpace(r.URL.Query().Get("cohort_start")), CohortEnd: strings.TrimSpace(r.URL.Query().Get("cohort_end"))})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) getGeo(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	resp, err := h.service.GetGeo(r.Context(), actor, application.DateRangeInput{StartDate: strings.TrimSpace(r.URL.Query().Get("start_date")), EndDate: strings.TrimSpace(r.URL.Query().Get("end_date"))})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) getForecast(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	resp, err := h.service.GetForecast(r.Context(), actor, application.LeaderboardInput{Period: strings.TrimSpace(r.URL.Query().Get("period"))})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) createExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp, err := h.service.RequestExport(r.Context(), actor, application.ExportInput{ExportType: strings.TrimSpace(req.ExportType), Period: strings.TrimSpace(req.Period), Format: strings.TrimSpace(req.Format), Filters: req.Filters})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusAccepted, "", application.MakeExportResponse(resp))
}

func (h *Handler) getExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	resp, err := h.service.GetExport(r.Context(), actor, chi.URLParam(r, "id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", application.MakeExportResponse(resp))
}
