package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/domain"
)

func (s *Service) HandleCanonicalEvent(_ context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if err := validatePartitionKeyInvariant(envelope, envelope.PartitionKeyPath); err != nil {
		return err
	}
	return domain.ErrUnsupportedEventType
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	if s.outbox == nil {
		return nil
	}
	pending, err := s.outbox.ListPending(ctx, s.cfg.OutboxFlushBatchSize)
	if err != nil {
		return err
	}
	for _, rec := range pending {
		now := s.nowFn()
		switch rec.EventClass {
		case domain.CanonicalEventClassDomain:
			if s.domainEvents != nil {
				if err := s.domainEvents.PublishDomain(ctx, rec.Envelope); err != nil {
					if s.dlq != nil {
						nowDLQ := s.nowFn()
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: nowDLQ, LastErrorAt: nowDLQ, SourceTopic: rec.Envelope.EventType, DLQTopic: "risk-service.dlq", TraceID: rec.Envelope.TraceID})
					}
					return err
				}
			}
		case domain.CanonicalEventClassAnalyticsOnly:
			if s.analytics != nil {
				_ = s.analytics.PublishAnalytics(ctx, rec.Envelope)
			}
		default:
			return fmt.Errorf("%w: %s", domain.ErrUnsupportedEventClass, rec.EventClass)
		}
		if err := s.outbox.MarkSent(ctx, rec.RecordID, now); err != nil {
			return err
		}
	}
	return nil
}

func publishDLQIdempotencyConflictRecord(key, source, traceID string, now time.Time) contracts.DLQRecord {
	return contracts.DLQRecord{OriginalEvent: contracts.EventEnvelope{EventID: uuid.NewString(), EventType: "risk.idempotency.conflict", EventClass: domain.CanonicalEventClassOps, OccurredAt: now, PartitionKeyPath: "envelope.source_service", PartitionKey: source, SourceService: source, TraceID: traceID, SchemaVersion: "v1", Data: []byte(`{"key":"` + key + `"}`)}, ErrorSummary: "idempotency key reused with mismatched payload", RetryCount: 1, FirstSeenAt: now, LastErrorAt: now, SourceTopic: "api", DLQTopic: "risk-service.dlq", TraceID: traceID}
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func validatePartitionKeyInvariant(event contracts.EventEnvelope, expectedPath string) error {
	if strings.TrimSpace(expectedPath) == "" || event.PartitionKeyPath != expectedPath {
		return domain.ErrInvalidEnvelope
	}
	if expectedPath == "envelope.source_service" {
		if event.PartitionKey != event.SourceService {
			return domain.ErrInvalidEnvelope
		}
		return nil
	}
	if !strings.HasPrefix(expectedPath, "data.") {
		return domain.ErrInvalidEnvelope
	}
	field := strings.TrimPrefix(expectedPath, "data.")
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return domain.ErrInvalidEnvelope
	}
	v, ok := payload[field]
	if !ok || fmt.Sprint(v) != event.PartitionKey {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func (s *Service) publishDLQIdempotencyConflict(ctx context.Context, key, traceID string) error {
	if s.dlq == nil {
		return nil
	}
	now := s.nowFn()
	return s.dlq.PublishDLQ(ctx, publishDLQIdempotencyConflictRecord(key, s.cfg.ServiceName, nonEmpty(traceID, uuid.NewString()), now))
}

var _ = time.RFC3339
