package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/domain"
)

type DeveloperRepository interface {
	Create(ctx context.Context, row domain.Developer) error
	GetByID(ctx context.Context, developerID string) (domain.Developer, error)
}

type SessionRepository interface {
	Create(ctx context.Context, row domain.DeveloperSession) error
}

type APIKeyRepository interface {
	Create(ctx context.Context, row domain.APIKey) error
	GetByID(ctx context.Context, keyID string) (domain.APIKey, error)
	Update(ctx context.Context, row domain.APIKey) error
}

type APIKeyRotationRepository interface {
	Create(ctx context.Context, row domain.APIKeyRotation) error
}

type WebhookRepository interface {
	Create(ctx context.Context, row domain.Webhook) error
	GetByID(ctx context.Context, webhookID string) (domain.Webhook, error)
}

type WebhookDeliveryRepository interface {
	Create(ctx context.Context, row domain.WebhookDelivery) error
}

type UsageRepository interface {
	CreateOrUpdate(ctx context.Context, row domain.DeveloperUsage) error
	GetByDeveloperID(ctx context.Context, developerID string) (domain.DeveloperUsage, error)
}

type AuditRepository interface {
	Append(ctx context.Context, row domain.AuditLog) error
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
