package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/ports"
)

func (s *Service) HandleCanonicalEvent(_ context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	return nil
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	if s.outbox == nil { return nil }
	pending, err := s.outbox.ListPending(ctx, s.cfg.OutboxFlushBatchSize)
	if err != nil { return err }
	for _, rec := range pending {
		now := s.nowFn()
		switch rec.EventClass {
		case domain.CanonicalEventClassDomain:
			if s.domainEvents != nil {
				if err := s.domainEvents.PublishDomain(ctx, rec.Envelope); err != nil {
					if s.dlq != nil {
						n := s.nowFn()
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: n, LastErrorAt: n, SourceTopic: rec.Envelope.EventType, DLQTopic: "escrow-ledger-service.dlq", TraceID: rec.Envelope.TraceID})
					}
					return err
				}
			}
		case domain.CanonicalEventClassAnalyticsOnly:
			if s.analytics != nil { _ = s.analytics.PublishAnalytics(ctx, rec.Envelope) }
		default:
			return fmt.Errorf("%w: %s", domain.ErrUnsupportedEventClass, rec.EventClass)
		}
		if err := s.outbox.MarkSent(ctx, rec.RecordID, now); err != nil { return err }
	}
	return nil
}

func (s *Service) enqueueEvent(ctx context.Context, eventType, traceID string, data any, escrowID string, now time.Time) error {
	if s.outbox == nil { return nil }
	if !domain.IsCanonicalEmittedEvent(eventType) { return domain.ErrUnsupportedEventType }
	b, err := json.Marshal(data)
	if err != nil { return domain.ErrInvalidInput }
	if strings.TrimSpace(traceID) == "" { traceID = uuid.NewString() }
	env := contracts.EventEnvelope{EventID: uuid.NewString(), EventType: eventType, EventClass: domain.CanonicalEventClass(eventType), OccurredAt: now, PartitionKeyPath: domain.CanonicalPartitionKeyPath(eventType), PartitionKey: escrowID, SourceService: s.cfg.ServiceName, TraceID: traceID, SchemaVersion: "v1", Data: b}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{RecordID: uuid.NewString(), EventClass: env.EventClass, Envelope: env, CreatedAt: now})
}

func (s *Service) enqueueHoldCreated(ctx context.Context, hold domain.EscrowHold, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventEscrowHoldCreated, traceID, contracts.EscrowHoldCreatedPayload{EscrowID: hold.EscrowID, CampaignID: hold.CampaignID, CreatorID: hold.CreatorID, Amount: hold.OriginalAmount, HeldAt: hold.HeldAt.UTC().Format(time.RFC3339)}, hold.EscrowID, now)
}
func (s *Service) enqueuePartialRelease(ctx context.Context, escrowID string, amount, remaining float64, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventEscrowPartialRelease, traceID, contracts.EscrowPartialReleasePayload{EscrowID: escrowID, Amount: amount, RemainingBalance: remaining, ReleasedAt: now.UTC().Format(time.RFC3339)}, escrowID, now)
}
func (s *Service) enqueueHoldFullyReleased(ctx context.Context, escrowID, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventEscrowHoldFullyReleased, traceID, contracts.EscrowHoldFullyReleasedPayload{EscrowID: escrowID, ReleasedAt: now.UTC().Format(time.RFC3339)}, escrowID, now)
}
func (s *Service) enqueueRefundProcessed(ctx context.Context, escrowID string, amount float64, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventEscrowRefundProcessed, traceID, contracts.EscrowRefundProcessedPayload{EscrowID: escrowID, Amount: amount, RefundedAt: now.UTC().Format(time.RFC3339)}, escrowID, now)
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() { return domain.ErrInvalidEnvelope }
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" { return domain.ErrInvalidEnvelope }
	if len(event.Data) == 0 { return domain.ErrInvalidEnvelope }
	return nil
}
