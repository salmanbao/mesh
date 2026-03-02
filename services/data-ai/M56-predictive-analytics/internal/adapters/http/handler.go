package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/contracts"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) getViewForecast(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	window, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("window_days")))
	forecast, err := h.service.GetViewForecast(r.Context(), actor, application.ViewForecastInput{
		UserID:     strings.TrimSpace(r.URL.Query().Get("user_id")),
		WindowDays: window,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "view forecast", forecast)
}

func (h *Handler) getClipRecommendations(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	items, err := h.service.GetClipRecommendations(r.Context(), actor, application.ClipRecommendationsInput{
		UserID: strings.TrimSpace(r.URL.Query().Get("user_id")),
		Limit:  limit,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "clip recommendations", map[string]interface{}{"items": items})
}

func (h *Handler) getChurnRisk(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	row, err := h.service.GetChurnRisk(r.Context(), actor, application.ChurnRiskInput{
		UserID: strings.TrimSpace(r.URL.Query().Get("user_id")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "churn risk", row)
}

func (h *Handler) predictCampaignSuccess(w http.ResponseWriter, r *http.Request) {
	var req contracts.CampaignSuccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	actor := actorFromContext(r.Context())
	row, err := h.service.PredictCampaignSuccess(r.Context(), actor, application.CampaignSuccessInput{
		CampaignID: strings.TrimSpace(r.PathValue("campaign_id")),
		RewardRate: req.RewardRate,
		Budget:     req.Budget,
		Niche:      strings.TrimSpace(req.Niche),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "campaign success prediction", row)
}
