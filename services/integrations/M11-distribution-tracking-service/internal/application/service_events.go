package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/ports"
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

func (s *Service) enqueueTrackingMetricsUpdated(ctx context.Context, snap domain.MetricSnapshot, post domain.TrackedPost, now time.Time) error {
	if s.outbox == nil {
		return nil
	}
	payload, _ := json.Marshal(contracts.TrackingMetricsUpdatedPayload{TrackedPostID: snap.TrackedPostID, Platform: snap.Platform, Views: snap.Views, Likes: snap.Likes, Shares: snap.Shares, Comments: snap.Comments, PolledAt: snap.PolledAt.UTC().Format(time.RFC3339)})
	env := contracts.EventEnvelope{EventID: uuid.NewString(), EventType: domain.EventTrackingMetricsUpdated, EventClass: domain.CanonicalEventClass(domain.EventTrackingMetricsUpdated), OccurredAt: now, PartitionKeyPath: domain.CanonicalPartitionKeyPath(domain.EventTrackingMetricsUpdated), PartitionKey: post.TrackedPostID, SourceService: s.cfg.ServiceName, TraceID: nonEmptyTrace("poll-" + post.TrackedPostID), SchemaVersion: "v1", Data: payload}
	rec := ports.OutboxRecord{RecordID: uuid.NewString(), EventClass: env.EventClass, Envelope: env, CreatedAt: now}
	return s.outbox.Enqueue(ctx, rec)
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
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: n, LastErrorAt: n, SourceTopic: rec.Envelope.EventType, DLQTopic: "distribution-tracking-service.dlq", TraceID: rec.Envelope.TraceID})
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

func nonEmptyTrace(v string) string {
	if strings.TrimSpace(v) == "" {
		return uuid.NewString()
	}
	return v
}
