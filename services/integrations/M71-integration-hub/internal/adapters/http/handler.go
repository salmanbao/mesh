package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/application"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) authorizeIntegration(w http.ResponseWriter, r *http.Request) {
	var req contracts.AuthorizeIntegrationRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	row, err := h.service.AuthorizeIntegration(r.Context(), actorFromContext(r.Context()), application.AuthorizeIntegrationInput{
		UserID:          req.UserID,
		IntegrationType: strings.TrimSpace(r.PathValue("type")),
		IntegrationName: req.IntegrationName,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "integration authorized", toIntegrationResponse(row))
}

func (h *Handler) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateWorkflow(r.Context(), actorFromContext(r.Context()), application.CreateWorkflowInput{
		UserID:              req.UserID,
		WorkflowName:        req.WorkflowName,
		WorkflowDescription: req.WorkflowDescription,
		TriggerEventType:    req.TriggerEventType,
		ActionType:          req.ActionType,
		IntegrationID:       req.IntegrationID,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "workflow created", toWorkflowResponse(row))
}

func (h *Handler) publishWorkflow(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.PublishWorkflow(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "workflow published", toWorkflowResponse(row))
}

func (h *Handler) testWorkflow(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.TestWorkflow(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "workflow test executed", toExecutionResponse(row))
}

func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context()))
		return
	}
	row, err := h.service.CreateWebhook(r.Context(), actorFromContext(r.Context()), application.CreateWebhookInput{
		UserID:      req.UserID,
		EndpointURL: req.EndpointURL,
		EventType:   req.EventType,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "webhook created", toWebhookResponse(row))
}

func (h *Handler) testWebhook(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.TestWebhook(r.Context(), actorFromContext(r.Context()), strings.TrimSpace(r.PathValue("id")))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "webhook test delivered", toWebhookDeliveryResponse(row))
}

func (h *Handler) chatPostMessage(w http.ResponseWriter, r *http.Request) {
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	ch, messageID, err := h.service.ChatPostMessage(r.Context(), actorFromContext(r.Context()), channel)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "message posted", contracts.ChatPostMessageResponse{
		Channel:   ch,
		MessageID: messageID,
		Status:    "queued",
	})
}

func toIntegrationResponse(row domain.Integration) contracts.IntegrationResponse {
	return contracts.IntegrationResponse{
		IntegrationID:   row.IntegrationID,
		UserID:          row.UserID,
		IntegrationType: row.IntegrationType,
		IntegrationName: row.IntegrationName,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toWorkflowResponse(row domain.Workflow) contracts.WorkflowResponse {
	return contracts.WorkflowResponse{
		WorkflowID:          row.WorkflowID,
		UserID:              row.UserID,
		WorkflowName:        row.WorkflowName,
		WorkflowDescription: row.WorkflowDescription,
		TriggerEventType:    row.TriggerEventType,
		ActionType:          row.ActionType,
		IntegrationID:       row.IntegrationID,
		Status:              row.Status,
		CreatedAt:           row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toExecutionResponse(row domain.WorkflowExecution) contracts.WorkflowExecutionResponse {
	return contracts.WorkflowExecutionResponse{
		ExecutionID: row.ExecutionID,
		WorkflowID:  row.WorkflowID,
		Status:      row.Status,
		TestRun:     row.TestRun,
		StartedAt:   row.StartedAt.UTC().Format(time.RFC3339),
	}
}

func toWebhookResponse(row domain.Webhook) contracts.WebhookResponse {
	return contracts.WebhookResponse{
		WebhookID:   row.WebhookID,
		UserID:      row.UserID,
		EndpointURL: row.EndpointURL,
		EventType:   row.EventType,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toWebhookDeliveryResponse(row domain.WebhookDelivery) contracts.WebhookDeliveryResponse {
	return contracts.WebhookDeliveryResponse{
		DeliveryID: row.DeliveryID,
		WebhookID:  row.WebhookID,
		Status:     row.Status,
		TestEvent:  row.TestEvent,
		CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
	}
}
