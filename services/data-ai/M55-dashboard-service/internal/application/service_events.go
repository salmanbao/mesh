package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/domain"
)

func (s *Service) HandleInternalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	now := s.nowFn()
	if envelope.EventID == "" {
		envelope.EventID = uuid.NewString()
	}
	duplicate, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, now)
	if err != nil {
		return err
	}
	if duplicate {
		return nil
	}
	if err := s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, now.Add(s.cfg.EventDedupTTL)); err != nil {
		return err
	}
	return domain.ErrUnsupportedEvent
}

func (s *Service) publishDLQIdempotencyConflict(ctx context.Context, key, traceID string) error {
	if s.dlq == nil {
		return nil
	}
	now := time.Now().UTC()
	return s.dlq.PublishDLQ(ctx, contracts.DLQRecord{
		OriginalEvent: contracts.EventEnvelope{
			EventID:          uuid.NewString(),
			EventType:        "",
			EventClass:       domain.CanonicalEventClassOps,
			OccurredAt:       now,
			PartitionKeyPath: "envelope.source_service",
			PartitionKey:     "M55-Dashboard-Service",
			SourceService:    "M55-Dashboard-Service",
			SchemaVersion:    "1.0",
			Metadata:         contracts.EnvelopeMetadata{TraceID: traceID},
			Data:             map[string]interface{}{"idempotency_key": key},
		},
		ErrorSummary: "idempotency key reused with mismatched payload",
		RetryCount:   1,
		FirstSeenAt:  now,
		LastErrorAt:  now,
		SourceTopic:  "api",
		TraceID:      traceID,
	})
}
