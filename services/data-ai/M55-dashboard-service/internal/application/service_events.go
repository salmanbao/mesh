package application

import (
	"context"
	"strings"
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
	switch strings.TrimSpace(envelope.EventType) {
	case "dashboard.cache_invalidation":
		userID := userIDFromEnvelope(envelope)
		if userID == "" {
			return domain.ErrInvalidInput
		}
		return s.cache.InvalidateByUser(ctx, userID)
	case "dashboard.real_time_update":
		// Best-effort transient update; no persistence requirement in this service.
		return nil
	default:
		return domain.ErrUnsupportedEvent
	}
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

func userIDFromEnvelope(envelope contracts.EventEnvelope) string {
	if payload, ok := envelope.Data.(map[string]interface{}); ok {
		if raw, ok := payload["user_id"]; ok {
			if userID, ok := raw.(string); ok {
				return strings.TrimSpace(userID)
			}
		}
	}
	return strings.TrimSpace(envelope.PartitionKey)
}
