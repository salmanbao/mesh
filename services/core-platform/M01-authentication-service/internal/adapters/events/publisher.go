package events

import (
	"context"
	"log/slog"
)

// LoggingPublisher is a development-friendly publisher implementation.
// It logs events instead of sending to a broker so early slices stay observable.
type LoggingPublisher struct {
	logger *slog.Logger
}

// NewLoggingPublisher creates a logging-based event publisher adapter.
func NewLoggingPublisher(logger *slog.Logger) *LoggingPublisher {
	return &LoggingPublisher{logger: logger}
}

func (p *LoggingPublisher) Publish(ctx context.Context, eventType string, payload []byte) error {
	p.logger.InfoContext(ctx, "event published",
		"module", "events.publisher",
		"layer", "adapter",
		"operation", "publish",
		"outcome", "success",
		"event_type", eventType,
		"payload_bytes", len(payload),
	)
	return nil
}
