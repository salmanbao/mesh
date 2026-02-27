package ports

import (
	"context"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/contracts"
)

type EventConsumer interface {
	Receive(ctx context.Context) (*contracts.EventEnvelope, error)
}

type DLQPublisher interface {
	PublishDLQ(ctx context.Context, record contracts.DLQRecord) error
}
