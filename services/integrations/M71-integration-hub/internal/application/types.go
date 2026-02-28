package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/ports"
)

type Config struct {
	ServiceName    string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type AuthorizeIntegrationInput struct {
	UserID          string
	IntegrationType string
	IntegrationName string
}

type CreateWorkflowInput struct {
	UserID              string
	WorkflowName        string
	WorkflowDescription string
	TriggerEventType    string
	ActionType          string
	IntegrationID       string
}

type CreateWebhookInput struct {
	UserID      string
	EndpointURL string
	EventType   string
}

type Service struct {
	cfg          Config
	integrations ports.IntegrationRepository
	credentials  ports.APICredentialRepository
	workflows    ports.WorkflowRepository
	executions   ports.WorkflowExecutionRepository
	webhooks     ports.WebhookRepository
	deliveries   ports.WebhookDeliveryRepository
	analytics    ports.AnalyticsRepository
	logs         ports.IntegrationLogRepository
	idempotency  ports.IdempotencyRepository
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Integrations ports.IntegrationRepository
	Credentials  ports.APICredentialRepository
	Workflows    ports.WorkflowRepository
	Executions   ports.WorkflowExecutionRepository
	Webhooks     ports.WebhookRepository
	Deliveries   ports.WebhookDeliveryRepository
	Analytics    ports.AnalyticsRepository
	Logs         ports.IntegrationLogRepository
	Idempotency  ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M71-Integration-Hub"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:          cfg,
		integrations: deps.Integrations,
		credentials:  deps.Credentials,
		workflows:    deps.Workflows,
		executions:   deps.Executions,
		webhooks:     deps.Webhooks,
		deliveries:   deps.Deliveries,
		analytics:    deps.Analytics,
		logs:         deps.Logs,
		idempotency:  deps.Idempotency,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}
