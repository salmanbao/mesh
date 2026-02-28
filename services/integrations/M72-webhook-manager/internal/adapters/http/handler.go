package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/application"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	wh, err := h.service.CreateWebhook(r.Context(), actor, application.CreateWebhookInput{
		EndpointURL:        req.EndpointURL,
		EventTypes:         req.EventTypes,
		BatchModeEnabled:   req.BatchModeEnabled,
		BatchSize:          req.BatchSize,
		BatchWindowSeconds: req.BatchWindowSeconds,
		RateLimitPerMinute: req.RateLimitPerMinute,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", wh)
}

func (h *Handler) updateWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	wh, err := h.service.UpdateWebhook(r.Context(), actor, r.PathValue("id"), application.UpdateWebhookInput{
		EventTypes:         req.EventTypes,
		BatchModeEnabled:   req.BatchModeEnabled,
		BatchSize:          req.BatchSize,
		BatchWindowSeconds: req.BatchWindowSeconds,
		RateLimitPerMinute: req.RateLimitPerMinute,
		Status:             req.Status,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", wh)
}

func (h *Handler) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	wh, err := h.service.DeleteWebhook(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", wh)
}

func (h *Handler) getWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	id := firstNonEmpty(r.PathValue("id"), r.PathValue("webhook_id"))
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
	var req contracts.TestWebhookRequest
	body, err := ioReadAllLimit(r.Body, 64*1024)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "unable to read request body", requestIDFromContext(r.Context()))
		return
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			req.Payload = json.RawMessage(body)
		}
	}
	id := firstNonEmpty(r.PathValue("id"), r.PathValue("webhook_id"))
	res, err := h.service.TestWebhook(r.Context(), actor, id, application.TestWebhookInput{Payload: json.RawMessage(marshalPayload(req.Payload))})
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
	items, err := h.service.ListDeliveries(r.Context(), actor, firstNonEmpty(r.PathValue("id"), r.PathValue("webhook_id")), limit)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", items)
}

func (h *Handler) analytics(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	data, err := h.service.GetAnalytics(r.Context(), actor, firstNonEmpty(r.PathValue("id"), r.PathValue("webhook_id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", data)
}

func (h *Handler) enableWebhook(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	id := firstNonEmpty(r.PathValue("id"), r.PathValue("webhook_id"))
	wh, err := h.service.EnableWebhook(r.Context(), actor, id)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", wh)
}

func (h *Handler) receiveCompatibilityWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := ioReadAllLimit(r.Body, 256*1024)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "unable to read request body", requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusAccepted, "", h.service.ReceiveCompatibilityWebhook(r.Context(), payload))
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, http.StatusOK, "ok", nil)
}

func marshalPayload(v any) []byte {
	if v == nil {
		return nil
	}
	raw, _ := json.Marshal(v)
	return raw
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
