package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/domain"
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
	expectedPath := domain.CanonicalPartitionKeyPath(envelope.EventType)
	if err := validatePartitionKeyInvariant(envelope, expectedPath); err != nil {
		return err
	}
	now := s.nowFn()
	dup, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, now)
	if err != nil {
		return err
	}
	if dup {
		return nil
	}
	if err := s.applyInboundEvent(ctx, envelope); err != nil {
		return err
	}
	return s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, now.Add(s.cfg.EventDedupTTL))
}

func (s *Service) applyInboundEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	scoreInput, err := mapInboundEventToScore(envelope)
	if err != nil {
		return err
	}
	_, err = s.scoreAndPersist(ctx, scoreInput, envelope.TraceID, true)
	return err
}

func mapInboundEventToScore(envelope contracts.EventEnvelope) (ScoreInput, error) {
	in := ScoreInput{EventID: envelope.EventID, EventType: envelope.EventType, TraceID: envelope.TraceID, OccurredAt: envelope.OccurredAt.Format(time.RFC3339), RawPayload: append([]byte(nil), envelope.Data...)}
	switch envelope.EventType {
	case domain.EventAffiliateClickTracked:
		var p contracts.AffiliateClickTrackedPayload
		if err := json.Unmarshal(envelope.Data, &p); err != nil {
			return ScoreInput{}, domain.ErrInvalidEnvelope
		}
		in.AffiliateID = strings.TrimSpace(p.AffiliateID)
		in.ClickIP = strings.TrimSpace(p.IPHash)
		in.ReferralToken = strings.TrimSpace(p.LinkID)
		in.Metadata = map[string]string{"referrer_url": strings.TrimSpace(p.ReferrerURL)}
	case domain.EventAffiliateAttributionCreate:
		var p contracts.AffiliateAttributionCreatedPayload
		if err := json.Unmarshal(envelope.Data, &p); err != nil {
			return ScoreInput{}, domain.ErrInvalidEnvelope
		}
		in.AffiliateID = strings.TrimSpace(p.AffiliateID)
		in.ConversionID = strings.TrimSpace(p.ConversionID)
		in.OrderID = strings.TrimSpace(p.OrderID)
		in.Amount = p.Amount
	case domain.EventTransactionSucceeded:
		var p contracts.TransactionSucceededPayload
		if err := json.Unmarshal(envelope.Data, &p); err != nil {
			return ScoreInput{}, domain.ErrInvalidEnvelope
		}
		in.EventID = nonEmptyString(in.EventID, strings.TrimSpace(p.TransactionID))
		in.EventType = envelope.EventType
		in.UserID = strings.TrimSpace(p.UserID)
		in.Amount = p.Amount
		in.OrderID = strings.TrimSpace(p.TransactionID)
		in.AffiliateID = nonEmptyString(in.AffiliateID, "unknown")
	case domain.EventUserRegistered:
		var p contracts.UserRegisteredPayload
		if err := json.Unmarshal(envelope.Data, &p); err != nil {
			return ScoreInput{}, domain.ErrInvalidEnvelope
		}
		in.UserID = strings.TrimSpace(p.UserID)
		in.AffiliateID = nonEmptyString(in.AffiliateID, "unknown")
	default:
		return ScoreInput{}, domain.ErrUnsupportedEventType
	}
	if strings.TrimSpace(in.AffiliateID) == "" {
		in.AffiliateID = "unknown"
	}
	return in, nil
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
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: nowDLQ, LastErrorAt: nowDLQ, SourceTopic: rec.Envelope.EventType, DLQTopic: "referral-fraud-detection-service.dlq", TraceID: rec.Envelope.TraceID})
					}
					return err
				}
			}
		case domain.CanonicalEventClassAnalyticsOnly:
			if s.analytics != nil {
				if err := s.analytics.PublishAnalytics(ctx, rec.Envelope); err != nil {
					return err
				}
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
	return contracts.DLQRecord{OriginalEvent: contracts.EventEnvelope{EventID: uuid.NewString(), EventType: "referral_fraud.idempotency.conflict", EventClass: domain.CanonicalEventClassOps, OccurredAt: now, PartitionKeyPath: "envelope.source_service", PartitionKey: source, SourceService: source, TraceID: traceID, SchemaVersion: "v1", Data: []byte(`{"key":"` + key + `"}`)}, ErrorSummary: "idempotency key reused with mismatched payload", RetryCount: 1, FirstSeenAt: now, LastErrorAt: now, SourceTopic: "api", DLQTopic: "referral-fraud-detection-service.dlq", TraceID: traceID}
}

func (s *Service) publishDLQIdempotencyConflict(ctx context.Context, key, traceID string) error {
	if s.dlq == nil {
		return nil
	}
	now := s.nowFn()
	trace := strings.TrimSpace(traceID)
	if trace == "" {
		trace = uuid.NewString()
	}
	return s.dlq.PublishDLQ(ctx, publishDLQIdempotencyConflictRecord(key, s.cfg.ServiceName, trace, now))
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

func nonEmptyString(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}
