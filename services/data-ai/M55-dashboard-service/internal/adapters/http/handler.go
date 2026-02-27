package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/contracts"
)

func (h *Handler) getDashboard(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	query := application.DashboardQueryInput{
		ViewID:     strings.TrimSpace(r.URL.Query().Get("view_id")),
		DateRange:  strings.TrimSpace(r.URL.Query().Get("date_range")),
		FromDate:   strings.TrimSpace(r.URL.Query().Get("from_date")),
		ToDate:     strings.TrimSpace(r.URL.Query().Get("to_date")),
		DeviceType: strings.TrimSpace(r.URL.Query().Get("device_type")),
		Timezone:   strings.TrimSpace(r.Header.Get("User-Timezone")),
	}
	dashboard, err := h.service.GetDashboard(r.Context(), actor, query)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.DashboardResponse{Dashboard: dashboard})
}

func (h *Handler) saveLayout(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.SaveLayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	items := make([]application.LayoutItemInput, 0, len(req.Widgets))
	for _, item := range req.Widgets {
		items = append(items, application.LayoutItemInput{
			WidgetID: strings.TrimSpace(item.WidgetID),
			Position: item.Position,
			Visible:  item.Visible,
			Size:     strings.TrimSpace(item.Size),
		})
	}
	layout, err := h.service.SaveLayout(r.Context(), actor, application.SaveLayoutInput{DeviceType: req.DeviceType, Items: items})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "Layout saved", layout)
}

func (h *Handler) createCustomView(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateCustomViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	view, err := h.service.CreateCustomView(r.Context(), actor, application.CreateCustomViewInput{
		ViewName:         strings.TrimSpace(req.ViewName),
		WidgetIDs:        req.WidgetIDs,
		DateRangeDefault: strings.TrimSpace(req.DateRangeDefault),
		SetAsDefault:     req.SetAsDefault,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateCustomViewResponse{ViewID: view.ViewID, ViewName: view.ViewName})
}

func (h *Handler) invalidateDashboard(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var payload struct {
		TriggerEvent string   `json:"trigger_event"`
		Widgets      []string `json:"widgets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.RecordCacheInvalidation(r.Context(), actor, payload.TriggerEvent, payload.Widgets)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "Cache invalidated", row)
}
