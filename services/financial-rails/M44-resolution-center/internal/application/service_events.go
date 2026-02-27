package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/ports"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	expectedClass := domain.CanonicalEventClass(envelope.EventType)
	if strings.TrimSpace(envelope.EventClass) != "" && envelope.EventClass != expectedClass {
		return domain.ErrUnsupportedEventClass
	}
	expectedPartitionPath := domain.CanonicalPartitionKeyPath(envelope.EventType)
	if err := validatePartitionKeyInvariant(envelope, expectedPartitionPath); err != nil {
		return err
	}

	now := s.nowFn()
	if s.eventDedup != nil {
		dup, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, now)
		if err != nil {
			return err
		}
		if dup {
			return nil
		}
	}

	switch envelope.EventType {
	case domain.EventSubmissionApproved:
		var payload contracts.SubmissionApprovedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if err := s.applyConsumedEvent(ctx, envelope.EventType, map[string]string{"entity_id": payload.SubmissionID, "user_id": payload.UserID}, now); err != nil {
			return err
		}
	case domain.EventPayoutFailed:
		var payload contracts.PayoutFailedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if err := s.applyConsumedEvent(ctx, envelope.EventType, map[string]string{"entity_id": payload.PayoutID, "user_id": payload.UserID}, now); err != nil {
			return err
		}
	default:
		return domain.ErrUnsupportedEventType
	}

	if s.eventDedup != nil {
		return s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, now.Add(s.cfg.EventDedupTTL))
	}
	return nil
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	if s.outbox == nil {
		return nil
	}
	pending, err := s.outbox.ListPending(ctx, s.cfg.OutboxFlushBatchSize)
	if err != nil {
		return err
	}
	for _, record := range pending {
		now := s.nowFn()
		switch record.EventClass {
		case domain.CanonicalEventClassDomain:
			if s.eventDedup != nil {
				dup, err := s.eventDedup.IsDuplicate(ctx, record.Envelope.EventID, now)
				if err != nil {
					return err
				}
				if dup {
					if err := s.outbox.MarkSent(ctx, record.RecordID, now); err != nil {
						return err
					}
					continue
				}
			}
			if s.domainEvents != nil {
				if err := s.domainEvents.PublishDomain(ctx, record.Envelope); err != nil {
					if s.dlq != nil {
						nowDLQ := s.nowFn()
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: record.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: nowDLQ, LastErrorAt: nowDLQ, SourceTopic: record.Envelope.EventType, DLQTopic: "resolution-center.dlq", TraceID: record.Envelope.TraceID})
					}
					return err
				}
			}
			if s.eventDedup != nil {
				if err := s.eventDedup.MarkProcessed(ctx, record.Envelope.EventID, record.Envelope.EventType, now.Add(s.cfg.EventDedupTTL)); err != nil {
					return err
				}
			}
		case domain.CanonicalEventClassAnalyticsOnly:
			if s.analytics != nil {
				if err := s.analytics.PublishAnalytics(ctx, record.Envelope); err != nil {
					// best-effort semantics; no DLQ for analytics_only
				}
			}
		default:
			return fmt.Errorf("%w: %s", domain.ErrUnsupportedEventClass, record.EventClass)
		}
		if err := s.outbox.MarkSent(ctx, record.RecordID, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enqueueDisputeCreated(ctx context.Context, actor Actor, dispute domain.Dispute) error {
	payload := contracts.DisputeCreatedPayload{
		DisputeID:  dispute.DisputeID,
		UserID:     dispute.UserID,
		EntityType: dispute.EntityType,
		EntityID:   dispute.EntityID,
		CreatedAt:  dispute.CreatedAt.Format(timeLayoutRFC3339),
	}
	return s.enqueueDomainEvent(ctx, domain.EventDisputeCreated, domain.CanonicalPartitionKeyPath(domain.EventDisputeCreated), dispute.DisputeID, actor.RequestID, payload)
}

func (s *Service) publishDisputeResolvedAnalytics(ctx context.Context, actor Actor, dispute domain.Dispute) error {
	payload := contracts.DisputeResolvedPayload{DisputeID: dispute.DisputeID, ResolvedAt: s.nowFn().Format(timeLayoutRFC3339), Resolution: dispute.ResolutionType}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env := contracts.EventEnvelope{
		EventID:          uuid.NewString(),
		EventType:        domain.EventDisputeResolved,
		EventClass:       domain.CanonicalEventClassAnalyticsOnly,
		OccurredAt:       s.nowFn(),
		PartitionKeyPath: domain.CanonicalPartitionKeyPath(domain.EventDisputeResolved),
		PartitionKey:     dispute.DisputeID,
		SourceService:    s.cfg.ServiceName,
		TraceID:          nonEmpty(actor.RequestID, uuid.NewString()),
		SchemaVersion:    "v1",
		Data:             data,
	}
	if err := validatePartitionKeyInvariant(env, env.PartitionKeyPath); err != nil {
		return err
	}
	if s.analytics == nil {
		return nil
	}
	if err := s.analytics.PublishAnalytics(ctx, env); err != nil {
		// analytics_only is best effort, no DLQ and no error propagation in workflow path
		return nil
	}
	return nil
}

func (s *Service) enqueueDomainEvent(ctx context.Context, eventType, partitionPath, partitionKey, traceID string, payload any) error {
	if s.outbox == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env := contracts.EventEnvelope{EventID: uuid.NewString(), EventType: eventType, EventClass: domain.CanonicalEventClassDomain, OccurredAt: s.nowFn(), PartitionKeyPath: partitionPath, PartitionKey: partitionKey, SourceService: s.cfg.ServiceName, TraceID: nonEmpty(traceID, uuid.NewString()), SchemaVersion: "v1", Data: data}
	if err := validatePartitionKeyInvariant(env, partitionPath); err != nil {
		return err
	}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{RecordID: uuid.NewString(), EventClass: domain.CanonicalEventClassDomain, Envelope: env, CreatedAt: s.nowFn()})
}

func (s *Service) publishDLQIdempotencyConflict(ctx context.Context, key, traceID string) error {
	if s.dlq == nil {
		return nil
	}
	now := s.nowFn()
	return s.dlq.PublishDLQ(ctx, contracts.DLQRecord{
		OriginalEvent: contracts.EventEnvelope{EventID: uuid.NewString(), EventType: "resolution-center.idempotency.conflict", EventClass: domain.CanonicalEventClassOps, OccurredAt: now, PartitionKeyPath: "envelope.source_service", PartitionKey: s.cfg.ServiceName, SourceService: s.cfg.ServiceName, TraceID: nonEmpty(traceID, uuid.NewString()), SchemaVersion: "v1", Data: []byte(`{"key":"` + key + `"}`)},
		ErrorSummary:  "idempotency key reused with mismatched payload",
		RetryCount:    1,
		FirstSeenAt:   now,
		LastErrorAt:   now,
		SourceTopic:   "api",
		DLQTopic:      "resolution-center.dlq",
		TraceID:       nonEmpty(traceID, uuid.NewString()),
	})
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
	value, ok := payload[field]
	if !ok || fmt.Sprint(value) != event.PartitionKey {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

const timeLayoutRFC3339 = time.RFC3339
