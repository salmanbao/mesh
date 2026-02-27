package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/ports"
)

type Worker struct {
	logger       *slog.Logger
	consumer     ports.EventConsumer
	dlqPublisher ports.DLQPublisher
	service      *application.Service
	pollInterval time.Duration
}

func NewWorker(logger *slog.Logger, consumer ports.EventConsumer, dlqPublisher ports.DLQPublisher, service *application.Service, pollInterval time.Duration) *Worker {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
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
			if w.service != nil {
				if err := w.service.FlushOutbox(ctx); err != nil {
					return err
				}
			}
			if w.consumer == nil || w.service == nil {
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
			if err := w.service.HandleCanonicalEvent(ctx, *event); err != nil {
				if event.EventClass == domain.CanonicalEventClassAnalyticsOnly {
					w.logger.WarnContext(ctx, "analytics-only event dropped", "event_type", event.EventType, "event_id", event.EventID, "error", err)
					continue
				}
				w.logger.ErrorContext(ctx, "canonical event failed", "event_type", event.EventType, "event_id", event.EventID, "error", err)
				if w.dlqPublisher != nil {
					now := time.Now().UTC()
					_ = w.dlqPublisher.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: *event, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: now, LastErrorAt: now, SourceTopic: event.EventType, DLQTopic: "risk-service.dlq", TraceID: event.TraceID})
				}
			}
		}
	}
}
