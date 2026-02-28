package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) AuthorizeIntegration(ctx context.Context, actor Actor, in AuthorizeIntegrationInput) (domain.Integration, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.Integration{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Integration{}, domain.ErrIdempotencyRequired
	}
	in.IntegrationType = strings.TrimSpace(in.IntegrationType)
	in.IntegrationName = strings.TrimSpace(in.IntegrationName)
	if in.IntegrationType == "" {
		return domain.Integration{}, domain.ErrInvalidInput
	}
	if in.IntegrationName == "" {
		in.IntegrationName = in.IntegrationType + " workspace"
	}
	requestHash := hashJSON(map[string]string{"op": "authorize_integration", "user_id": userID, "integration_type": in.IntegrationType, "integration_name": in.IntegrationName})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Integration{}, err
	} else if ok {
		var out domain.Integration
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Integration{}, err
	}

	row := domain.Integration{
		IntegrationID:   nextID("int"),
		UserID:          userID,
		IntegrationType: in.IntegrationType,
		IntegrationName: in.IntegrationName,
		Status:          domain.IntegrationStatusConnected,
		CreatedAt:       s.nowFn(),
	}
	if err := s.integrations.Create(ctx, row); err != nil {
		return domain.Integration{}, err
	}
	if s.credentials != nil {
		_ = s.credentials.Create(ctx, domain.APICredential{
			CredentialID:   nextID("cred"),
			IntegrationID:  row.IntegrationID,
			CredentialType: "oauth_token",
			MaskedValue:    "enc_****",
			CreatedAt:      s.nowFn(),
		})
	}
	if s.analytics != nil {
		_ = s.analytics.CreateOrUpdate(ctx, domain.Analytics{
			AnalyticsID:      nextID("an"),
			IntegrationID:    row.IntegrationID,
			TotalEvents:      0,
			FailedEvents:     0,
			AggregationStart: s.nowFn().Truncate(24 * time.Hour),
		})
	}
	s.appendLog(ctx, row.IntegrationID, "connected", "success")
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) CreateWorkflow(ctx context.Context, actor Actor, in CreateWorkflowInput) (domain.Workflow, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.Workflow{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Workflow{}, domain.ErrIdempotencyRequired
	}
	in.WorkflowName = strings.TrimSpace(in.WorkflowName)
	in.TriggerEventType = strings.TrimSpace(in.TriggerEventType)
	in.ActionType = strings.TrimSpace(in.ActionType)
	in.WorkflowDescription = strings.TrimSpace(in.WorkflowDescription)
	in.IntegrationID = strings.TrimSpace(in.IntegrationID)
	if in.WorkflowName == "" || in.TriggerEventType == "" || in.ActionType == "" {
		return domain.Workflow{}, domain.ErrInvalidInput
	}
	if in.IntegrationID != "" {
		intg, err := s.integrations.GetByID(ctx, in.IntegrationID)
		if err != nil {
			return domain.Workflow{}, err
		}
		if intg.UserID != userID {
			return domain.Workflow{}, domain.ErrForbidden
		}
	}
	requestHash := hashJSON(map[string]string{"op": "create_workflow", "user_id": userID, "workflow_name": in.WorkflowName, "trigger": in.TriggerEventType, "action": in.ActionType, "integration_id": in.IntegrationID})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Workflow{}, err
	} else if ok {
		var out domain.Workflow
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Workflow{}, err
	}

	row := domain.Workflow{
		WorkflowID:          nextID("wf"),
		UserID:              userID,
		WorkflowName:        in.WorkflowName,
		WorkflowDescription: in.WorkflowDescription,
		TriggerEventType:    in.TriggerEventType,
		ActionType:          in.ActionType,
		IntegrationID:       in.IntegrationID,
		Status:              domain.WorkflowStatusDraft,
		CreatedAt:           s.nowFn(),
	}
	if err := s.workflows.Create(ctx, row); err != nil {
		return domain.Workflow{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) PublishWorkflow(ctx context.Context, actor Actor, workflowID string) (domain.Workflow, error) {
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Workflow{}, domain.ErrIdempotencyRequired
	}
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return domain.Workflow{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "publish_workflow", "workflow_id": workflowID})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Workflow{}, err
	} else if ok {
		var out domain.Workflow
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Workflow{}, err
	}
	row, err := s.workflows.GetByID(ctx, workflowID)
	if err != nil {
		return domain.Workflow{}, err
	}
	if !canAccessUser(actor, row.UserID) {
		return domain.Workflow{}, authorizeError(actor)
	}
	row.Status = domain.WorkflowStatusPublished
	if err := s.workflows.Update(ctx, row); err != nil {
		return domain.Workflow{}, err
	}
	s.appendLog(ctx, row.IntegrationID, "workflow_published", "success")
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) TestWorkflow(ctx context.Context, actor Actor, workflowID string) (domain.WorkflowExecution, error) {
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return domain.WorkflowExecution{}, domain.ErrInvalidInput
	}
	row, err := s.workflows.GetByID(ctx, workflowID)
	if err != nil {
		return domain.WorkflowExecution{}, err
	}
	if !canAccessUser(actor, row.UserID) {
		return domain.WorkflowExecution{}, authorizeError(actor)
	}
	exec := domain.WorkflowExecution{
		ExecutionID: nextID("exec"),
		WorkflowID:  workflowID,
		Status:      domain.ExecutionStatusSuccess,
		TestRun:     true,
		StartedAt:   s.nowFn(),
	}
	if err := s.executions.Create(ctx, exec); err != nil {
		return domain.WorkflowExecution{}, err
	}
	s.appendLog(ctx, row.IntegrationID, "workflow_executed", "success")
	return exec, nil
}

func (s *Service) CreateWebhook(ctx context.Context, actor Actor, in CreateWebhookInput) (domain.Webhook, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.Webhook{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Webhook{}, domain.ErrIdempotencyRequired
	}
	in.EndpointURL = strings.TrimSpace(in.EndpointURL)
	in.EventType = strings.TrimSpace(in.EventType)
	if in.EndpointURL == "" || in.EventType == "" {
		return domain.Webhook{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "create_webhook", "user_id": userID, "endpoint_url": in.EndpointURL, "event_type": in.EventType})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	} else if ok {
		var out domain.Webhook
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	}
	row := domain.Webhook{
		WebhookID:   nextID("wh"),
		UserID:      userID,
		EndpointURL: in.EndpointURL,
		EventType:   in.EventType,
		Status:      domain.WebhookStatusActive,
		CreatedAt:   s.nowFn(),
	}
	if err := s.webhooks.Create(ctx, row); err != nil {
		return domain.Webhook{}, err
	}
	s.appendLog(ctx, "", "webhook_created", "success")
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) TestWebhook(ctx context.Context, actor Actor, webhookID string) (domain.WebhookDelivery, error) {
	webhookID = strings.TrimSpace(webhookID)
	if webhookID == "" {
		return domain.WebhookDelivery{}, domain.ErrInvalidInput
	}
	wh, err := s.webhooks.GetByID(ctx, webhookID)
	if err != nil {
		return domain.WebhookDelivery{}, err
	}
	if !canAccessUser(actor, wh.UserID) {
		return domain.WebhookDelivery{}, authorizeError(actor)
	}
	row := domain.WebhookDelivery{
		DeliveryID: nextID("delivery"),
		WebhookID:  webhookID,
		Status:     domain.ExecutionStatusSuccess,
		TestEvent:  true,
		CreatedAt:  s.nowFn(),
	}
	if err := s.deliveries.Create(ctx, row); err != nil {
		return domain.WebhookDelivery{}, err
	}
	return row, nil
}

func (s *Service) ChatPostMessage(ctx context.Context, actor Actor, channel string) (string, string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", "", domain.ErrUnauthorized
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "#general"
	}
	return channel, nextID("msg"), nil
}

func (s *Service) resolveUser(actor Actor, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if requested == "" {
		return "", domain.ErrUnauthorized
	}
	if !canAccessUser(actor, requested) {
		return "", authorizeError(actor)
	}
	return requested, nil
}

func canAccessUser(actor Actor, userID string) bool {
	if strings.TrimSpace(actor.SubjectID) == "" || strings.TrimSpace(userID) == "" {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	return actor.SubjectID == userID || role == "admin" || role == "support"
}

func authorizeError(actor Actor) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ErrUnauthorized
	}
	return domain.ErrForbidden
}

func (s *Service) appendLog(ctx context.Context, integrationID, actionType, status string) {
	if s.logs == nil {
		return
	}
	_ = s.logs.Append(ctx, domain.IntegrationLog{
		LogID:         nextID("log"),
		IntegrationID: integrationID,
		ActionType:    actionType,
		Status:        status,
		ActionAt:      s.nowFn(),
	})
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) getIdempotent(ctx context.Context, key, expectedHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != expectedHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return rec.ResponseBody, true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, v any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(v)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}
