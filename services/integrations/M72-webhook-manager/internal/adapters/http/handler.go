package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/application"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req application.CreateWebhookInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	wh, err := h.service.CreateWebhook(r.Context(), actor, req)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", wh)
}

func (h *Handler) getWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	id := chi.URLParam(r, "webhook_id")
	wh, err := h.service.GetWebhook(r.Context(), actor, id)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", wh)
}

func (h *Handler) testWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	payload, _ := ioReadAllLimit(r.Body, 64*1024)
	id := chi.URLParam(r, "webhook_id")
	res, err := h.service.TestWebhook(r.Context(), actor, id, payload)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", res)
}

func (h *Handler) listDeliveries(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	items, err := h.service.ListDeliveries(r.Context(), actor, chi.URLParam(r, "webhook_id"), limit)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", items)
}

func (h *Handler) analytics(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	data, err := h.service.GetAnalytics(r.Context(), actor, chi.URLParam(r, "webhook_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", data)
}

func (h *Handler) enableWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	id := chi.URLParam(r, "webhook_id")
	wh, err := h.service.EnableWebhook(r.Context(), actor, id)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", wh)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, "", map[string]string{"status": "ok"})
}
