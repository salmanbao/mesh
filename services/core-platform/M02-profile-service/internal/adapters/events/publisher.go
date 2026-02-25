package events

import (
	"context"
	"log/slog"
)

type LoggingPublisher struct {
	logger *slog.Logger
}

func NewLoggingPublisher(logger *slog.Logger) *LoggingPublisher {
	return &LoggingPublisher{logger: logger}
}

func (p *LoggingPublisher) Publish(ctx context.Context, eventType string, payload []byte, partitionKey string) error {
	p.logger.InfoContext(ctx, "event published",
		"module", "events.publisher",
		"layer", "adapter",
		"operation", "publish",
		"outcome", "success",
		"event_type", eventType,
		"partition_key", partitionKey,
		"payload_bytes", len(payload),
	)
	return nil
}
