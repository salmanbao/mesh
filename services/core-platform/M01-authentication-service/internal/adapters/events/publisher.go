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

func (p *LoggingPublisher) Publish(ctx context.Context, eventType string, payload []byte) error {
	p.logger.InfoContext(ctx, "published event", "event_type", eventType, "payload", string(payload))
	return nil
}
