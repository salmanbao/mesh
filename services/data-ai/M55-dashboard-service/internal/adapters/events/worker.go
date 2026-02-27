package events

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/ports"
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
			if err := w.service.HandleInternalEvent(ctx, *event); err != nil {
				now := time.Now().UTC()
				_ = w.dlqPublisher.PublishDLQ(ctx, contracts.DLQRecord{
					OriginalEvent: *event,
					ErrorSummary:  err.Error(),
					RetryCount:    1,
					FirstSeenAt:   now,
					LastErrorAt:   now,
					SourceTopic:   event.EventType,
				})
				w.logger.ErrorContext(ctx, "internal event failed", "event_type", event.EventType, "event_id", event.EventID, "error", err)
			}
		}
	}
}
