package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/domain"
)

type IntegrationRepository interface {
	Create(ctx context.Context, row domain.Integration) error
	GetByID(ctx context.Context, integrationID string) (domain.Integration, error)
}

type APICredentialRepository interface {
	Create(ctx context.Context, row domain.APICredential) error
}

type WorkflowRepository interface {
	Create(ctx context.Context, row domain.Workflow) error
	GetByID(ctx context.Context, workflowID string) (domain.Workflow, error)
	Update(ctx context.Context, row domain.Workflow) error
}

type WorkflowExecutionRepository interface {
	Create(ctx context.Context, row domain.WorkflowExecution) error
}

type WebhookRepository interface {
	Create(ctx context.Context, row domain.Webhook) error
	GetByID(ctx context.Context, webhookID string) (domain.Webhook, error)
}

type WebhookDeliveryRepository interface {
	Create(ctx context.Context, row domain.WebhookDelivery) error
}

type AnalyticsRepository interface {
	CreateOrUpdate(ctx context.Context, row domain.Analytics) error
}

type IntegrationLogRepository interface {
	Append(ctx context.Context, row domain.IntegrationLog) error
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}
