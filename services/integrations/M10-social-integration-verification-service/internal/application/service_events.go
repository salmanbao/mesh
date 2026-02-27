package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/ports"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
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
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: n, LastErrorAt: n, SourceTopic: rec.Envelope.EventType, DLQTopic: "social-integration-verification-service.dlq", TraceID: rec.Envelope.TraceID})
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

func (s *Service) enqueueDomainEvent(ctx context.Context, eventType, traceID string, data any, partitionKey string, now time.Time) error {
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
	env := contracts.EventEnvelope{
		EventID:          uuid.NewString(),
		EventType:        eventType,
		EventClass:       domain.CanonicalEventClass(eventType),
		OccurredAt:       now,
		PartitionKeyPath: domain.CanonicalPartitionKeyPath(eventType),
		PartitionKey:     partitionKey,
		SourceService:    s.cfg.ServiceName,
		TraceID:          traceID,
		SchemaVersion:    "v1",
		Data:             b,
	}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{RecordID: uuid.NewString(), EventClass: env.EventClass, Envelope: env, CreatedAt: now})
}

func (s *Service) enqueueSocialAccountConnected(ctx context.Context, acc domain.SocialAccount, traceID string, now time.Time) error {
	return s.enqueueDomainEvent(ctx, domain.EventSocialAccountConnected, traceID, contracts.SocialAccountConnectedPayload{UserID: acc.UserID, Platform: acc.Provider, ConnectedAt: acc.ConnectedAt.UTC().Format(time.RFC3339)}, acc.UserID, now)
}
func (s *Service) enqueueSocialStatusChanged(ctx context.Context, userID, platform, status, traceID string, now time.Time) error {
	return s.enqueueDomainEvent(ctx, domain.EventSocialStatusChanged, traceID, contracts.SocialStatusChangedPayload{UserID: userID, Platform: platform, Status: status, ChangedAt: now.UTC().Format(time.RFC3339)}, userID, now)
}
func (s *Service) enqueueFollowersSynced(ctx context.Context, metric domain.SocialMetric, traceID string, now time.Time) error {
	return s.enqueueDomainEvent(ctx, domain.EventSocialFollowersSynced, traceID, contracts.SocialFollowersSyncedPayload{UserID: metric.UserID, Platform: metric.Provider, FollowerCount: metric.FollowerCount, SyncedAt: metric.SyncedAt.UTC().Format(time.RFC3339)}, metric.UserID, now)
}
func (s *Service) enqueuePostValidated(ctx context.Context, input PostValidationInput, traceID string, now time.Time) error {
	return s.enqueueDomainEvent(ctx, domain.EventSocialPostValidated, traceID, contracts.SocialPostValidatedPayload{UserID: input.UserID, Platform: input.Platform, PostID: input.PostID, ValidatedAt: now.UTC().Format(time.RFC3339)}, input.UserID, now)
}
func (s *Service) enqueueComplianceViolation(ctx context.Context, input ComplianceViolationInput, traceID string, now time.Time) error {
	return s.enqueueDomainEvent(ctx, domain.EventSocialComplianceViolation, traceID, contracts.SocialComplianceViolationPayload{UserID: input.UserID, Platform: input.Platform, PostID: input.PostID, ViolationAt: now.UTC().Format(time.RFC3339), Reason: input.Reason}, input.UserID, now)
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
