package ports

import (
	"context"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/contracts"
)

type EventConsumer interface {
	Receive(ctx context.Context) (*contracts.EventEnvelope, error)
}
type DomainPublisher interface {
	PublishDomain(ctx context.Context, event contracts.EventEnvelope) error
}
type AnalyticsPublisher interface {
	PublishAnalytics(ctx context.Context, event contracts.EventEnvelope) error
}
type DLQPublisher interface {
	PublishDLQ(ctx context.Context, record contracts.DLQRecord) error
}
