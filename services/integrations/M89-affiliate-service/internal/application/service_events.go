package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/ports"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if s.eventDedup != nil {
		dup, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, s.nowFn())
		if err != nil {
			return err
		}
		if dup {
			return nil
		}
		if err := s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, s.nowFn().Add(s.cfg.EventDedupTTL)); err != nil {
			return err
		}
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
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
	for _, rec := range pending {
		now := s.nowFn()
		switch rec.EventClass {
		case domain.CanonicalEventClassDomain:
			if s.domainEvents != nil {
				if err := s.domainEvents.PublishDomain(ctx, rec.Envelope); err != nil {
					if s.dlq != nil {
						n := s.nowFn()
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: n, LastErrorAt: n, SourceTopic: rec.Envelope.EventType, DLQTopic: "affiliate-service.dlq", TraceID: rec.Envelope.TraceID})
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

func (s *Service) enqueueEvent(ctx context.Context, eventType, traceID string, data any, affiliateID string, now time.Time) error {
	if s.outbox == nil {
		return nil
	}
	if !domain.IsCanonicalEmittedEvent(eventType) {
		return domain.ErrUnsupportedEventType
	}
	b, err := json.Marshal(data)
	if err != nil {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = uuid.NewString()
	}
	env := contracts.EventEnvelope{EventID: uuid.NewString(), EventType: eventType, EventClass: domain.CanonicalEventClass(eventType), OccurredAt: now, PartitionKeyPath: domain.CanonicalPartitionKeyPath(eventType), PartitionKey: affiliateID, SourceService: s.cfg.ServiceName, TraceID: traceID, SchemaVersion: "v1", Data: b}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{RecordID: uuid.NewString(), EventClass: env.EventClass, Envelope: env, CreatedAt: now})
}

func (s *Service) enqueueAffiliateClickTracked(ctx context.Context, click domain.ReferralClick, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventAffiliateClickTracked, traceID, contracts.AffiliateClickTrackedPayload{AffiliateID: click.AffiliateID, LinkID: click.LinkID, ReferrerURL: click.ReferrerURL, IPHash: click.IPHash, TrackedAt: click.ClickedAt.UTC().Format(time.RFC3339)}, click.AffiliateID, now)
}
func (s *Service) enqueueAffiliateAttributionCreated(ctx context.Context, attr domain.ReferralAttribution, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventAffiliateAttributionCreated, traceID, contracts.AffiliateAttributionCreatedPayload{AffiliateID: attr.AffiliateID, ConversionID: attr.ConversionID, OrderID: attr.OrderID, Amount: attr.Amount, Currency: attr.Currency, AttributedAt: attr.AttributedAt.UTC().Format(time.RFC3339)}, attr.AffiliateID, now)
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.PartitionKeyPath) == "" || strings.TrimSpace(event.PartitionKey) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}
