package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/ports"
)

type Worker struct {
	logger       *slog.Logger
	consumer     ports.EventConsumer
	dlqPublisher ports.DLQPublisher
	service      *application.Service
	pollInterval time.Duration
}

func NewWorker(logger *slog.Logger, consumer ports.EventConsumer, dlqPublisher ports.DLQPublisher, service *application.Service, pollInterval time.Duration) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &Worker{logger: logger, consumer: consumer, dlqPublisher: dlqPublisher, service: service, pollInterval: pollInterval}
}
func (w *Worker) Run(ctx context.Context) error {
	t := time.NewTicker(w.pollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if w.service != nil {
				if err := w.service.FlushOutbox(ctx); err != nil {
					return err
				}
			}
			if w.consumer == nil || w.service == nil {
				continue
			}
			e, err := w.consumer.Receive(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					continue
				}
				return err
			}
			if e == nil {
				continue
			}
			if err := w.service.HandleCanonicalEvent(ctx, *e); err != nil {
				if e.EventClass == domain.CanonicalEventClassAnalyticsOnly {
					w.logger.WarnContext(ctx, "analytics-only event dropped", "event_type", e.EventType, "event_id", e.EventID, "error", err)
					continue
				}
				w.logger.ErrorContext(ctx, "canonical event failed", "event_type", e.EventType, "event_id", e.EventID, "error", err)
				if w.dlqPublisher != nil {
					now := time.Now().UTC()
					_ = w.dlqPublisher.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: *e, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: now, LastErrorAt: now, SourceTopic: e.EventType, DLQTopic: "social-integration-verification-service.dlq", TraceID: e.TraceID})
				}
			}
		}
	}
}
