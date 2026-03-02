package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/ports"
)

type Worker struct {
	logger       *slog.Logger
	consumer     ports.EventConsumer
	dlqPublisher ports.DLQPublisher
	service      *application.Service
	pollInterval time.Duration
}

func NewWorker(logger *slog.Logger, consumer ports.EventConsumer, dlqPublisher ports.DLQPublisher, service *application.Service, pollInterval time.Duration) *Worker {
	return &Worker{logger: logger, consumer: consumer, dlqPublisher: dlqPublisher, service: service, pollInterval: pollInterval}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.service.FlushOutbox(ctx); err != nil {
				return err
			}
			if w.consumer == nil {
				continue
			}
			event, err := w.consumer.Receive(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					continue
				}
				return err
			}
			if event == nil {
				continue
			}
			if err := w.service.HandleInboundEvent(ctx, *event); err != nil {
				if errors.Is(err, domain.ErrUnsupportedEventType) {
					// M58 has no canonical inbound event dependencies; ignore unknown bus traffic.
					w.logger.WarnContext(ctx, "unsupported inbound event ignored", "event_type", event.EventType, "event_id", event.EventID)
					continue
				}
				if event.EventClass == domain.CanonicalEventClassAnalyticsOnly || domain.CanonicalEventClass(event.EventType) == domain.CanonicalEventClassAnalyticsOnly {
					w.logger.WarnContext(ctx, "analytics-only event dropped", "event_type", event.EventType, "event_id", event.EventID, "error", err)
					continue
				}
				now := time.Now().UTC()
				_ = w.dlqPublisher.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: *event, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: now, LastErrorAt: now, SourceTopic: event.EventType, DLQTopic: "recommendation-service.dlq", TraceID: event.TraceID})
				w.logger.ErrorContext(ctx, "event routed to dlq", "event_type", event.EventType, "event_id", event.EventID, "error", err)
			}
		}
	}
}
