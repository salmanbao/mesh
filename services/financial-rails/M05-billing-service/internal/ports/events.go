package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/contracts"
)

type EventConsumer interface {
	Receive(ctx context.Context) (*contracts.EventEnvelope, error)
}

type DLQPublisher interface {
	Publish(ctx context.Context, record contracts.DLQRecord) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}
