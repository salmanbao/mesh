package events

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

type OutboxWorker struct {
	logger    *slog.Logger
	outbox    ports.OutboxRepository
	publisher ports.EventPublisher
	interval  time.Duration
	batchSize int
}

func NewOutboxWorker(logger *slog.Logger, outbox ports.OutboxRepository, publisher ports.EventPublisher, interval time.Duration, batchSize int) *OutboxWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &OutboxWorker{
		logger: logger, outbox: outbox, publisher: publisher, interval: interval, batchSize: batchSize,
	}
}

func (w *OutboxWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := w.processOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.ErrorContext(ctx, "outbox iteration failed",
				"module", "events.outbox_worker",
				"layer", "adapter",
				"operation", "process_once",
				"outcome", "failure",
				"error", err,
			)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *OutboxWorker) processOnce(ctx context.Context) error {
	records, err := w.outbox.FetchUnpublished(ctx, w.batchSize)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, rec := range records {
		if err := w.publisher.Publish(ctx, rec.EventType, rec.Payload, rec.PartitionKey); err != nil {
			_ = w.outbox.MarkFailed(ctx, rec.OutboxID, err.Error(), now)
			continue
		}
		_ = w.outbox.MarkPublished(ctx, rec.OutboxID, now)
	}
	return nil
}
