package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/domain"
)

type WebhookRepository interface {
	Create(ctx context.Context, wh domain.Webhook) error
	Update(ctx context.Context, wh domain.Webhook) error
	Get(ctx context.Context, id string) (domain.Webhook, error)
}

type DeliveryRepository interface {
	Add(ctx context.Context, d domain.Delivery) error
	ListByWebhook(ctx context.Context, webhookID string, limit int) ([]domain.Delivery, error)
}

type AnalyticsRepository interface {
	Snapshot(ctx context.Context, webhookID string) (domain.Analytics, error)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error)
	Upsert(ctx context.Context, rec domain.IdempotencyRecord) error
}
