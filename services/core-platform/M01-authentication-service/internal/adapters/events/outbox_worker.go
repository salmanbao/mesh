package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// OutboxWorker pulls unpublished outbox records and publishes them.
// This separates transactional writes from broker delivery for reliability.
type OutboxWorker struct {
	logger     *slog.Logger
	outbox     ports.OutboxRepository
	publisher  ports.EventPublisher
	interval   time.Duration
	batchSize  int
	claimTTL   time.Duration
	maxRetries int
}

// NewOutboxWorker constructs the outbox publisher loop with sane defaults.
func NewOutboxWorker(
	logger *slog.Logger,
	outbox ports.OutboxRepository,
	publisher ports.EventPublisher,
	interval time.Duration,
	batchSize int,
	claimTTL time.Duration,
	maxRetries int,
) *OutboxWorker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if claimTTL <= 0 {
		claimTTL = 30 * time.Second
	}
	if maxRetries <= 0 {
		maxRetries = 5
	}
	return &OutboxWorker{
		logger:     logger,
		outbox:     outbox,
		publisher:  publisher,
		interval:   interval,
		batchSize:  batchSize,
		claimTTL:   claimTTL,
		maxRetries: maxRetries,
	}
}

// Run executes the periodic outbox publish loop until context cancellation.
func (w *OutboxWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.processOnce(ctx); err != nil {
			w.logger.ErrorContext(ctx, "outbox iteration failed",
				"module", "events.outbox_worker",
				"layer", "adapter",
				"operation", "outbox_process_once",
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
	claimToken := uuid.NewString()
	records, err := w.outbox.ClaimUnpublished(ctx, w.batchSize, claimToken, time.Now().UTC().Add(w.claimTTL))
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	published := 0
	failed := 0
	deadLettered := 0
	for _, rec := range records {
		if rec.RetryCount >= w.maxRetries {
			deadLettered++
			_ = w.outbox.MarkDeadLettered(ctx, rec.OutboxID, claimToken, "retry threshold reached before publish", now)
			continue
		}

		if err := w.publisher.Publish(ctx, rec.EventType, rec.Payload); err != nil {
			failed++
			retriesAfterFailure := rec.RetryCount + 1
			if retriesAfterFailure >= w.maxRetries {
				deadLettered++
				w.logger.ErrorContext(ctx, "outbox message moved to dlq",
					"module", "events.outbox_worker",
					"layer", "adapter",
					"operation", "publish_event",
					"outcome", "failure",
					"outbox_id", rec.OutboxID,
					"event_type", rec.EventType,
					"payload_bytes", len(rec.Payload),
					"retry_count", retriesAfterFailure,
					"error", err,
				)
				_ = w.outbox.MarkDeadLettered(ctx, rec.OutboxID, claimToken, err.Error(), now)
				continue
			}

			w.logger.WarnContext(ctx, "outbox publish failed; retry scheduled",
				"module", "events.outbox_worker",
				"layer", "adapter",
				"operation", "publish_event",
				"outcome", "failure",
				"outbox_id", rec.OutboxID,
				"event_type", rec.EventType,
				"payload_bytes", len(rec.Payload),
				"retry_count", retriesAfterFailure,
				"error", err,
			)
			_ = w.outbox.MarkFailed(ctx, rec.OutboxID, claimToken, err.Error(), now)
			continue
		}
		published++
		_ = w.outbox.MarkPublished(ctx, rec.OutboxID, claimToken, now)
	}
	if len(records) > 0 {
		w.logger.InfoContext(ctx, "outbox batch processed",
			"module", "events.outbox_worker",
			"layer", "adapter",
			"operation", "outbox_process_once",
			"outcome", "success",
			"batch_size", len(records),
			"published_count", published,
			"failed_count", failed,
			"dead_lettered_count", deadLettered,
		)
	}
	return nil
}
