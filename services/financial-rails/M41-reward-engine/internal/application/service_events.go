package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/ports"
)

func (s *Service) HandleDomainEvent(ctx context.Context, event contracts.EventEnvelope) error {
	if !s.cfg.EnableDomainEventConsumption {
		return nil
	}
	if !isSupportedEventType(event.EventType) {
		return domain.ErrUnsupportedEventType
	}
	if event.EventClass != "" && event.EventClass != domain.CanonicalEventClassDomain {
		return domain.ErrUnsupportedEventClass
	}

	allowedPartitionPaths := []string{"data.submission_id", "submission_id"}
	if event.EventType == domain.EventTrackingMetricsUpdated {
		allowedPartitionPaths = []string{"data.tracked_post_id", "tracked_post_id"}
	}
	if err := validateDomainEventEnvelope(event, allowedPartitionPaths...); err != nil {
		return err
	}

	now := s.nowFn()
	dup, err := s.eventDedup.IsDuplicate(ctx, event.EventID, now)
	if err != nil {
		return err
	}
	if dup {
		return nil
	}

	switch event.EventType {
	case domain.EventSubmissionAutoApproved:
		var payload contracts.SubmissionAutoApprovedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode submission.auto_approved payload: %w", err)
		}
		_, err = s.calculateWithKey(ctx, CalculateRewardInput{
			UserID:       payload.UserID,
			SubmissionID: payload.SubmissionID,
			CampaignID:   payload.CampaignID,
			EventID:      event.EventID,
		}, "event:"+event.EventID)
	case domain.EventSubmissionVerified:
		var payload contracts.SubmissionVerifiedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode submission.verified payload: %w", err)
		}
		verifiedAt := now
		if parsed, parseErr := time.Parse(time.RFC3339, payload.VerifiedAt); parseErr == nil {
			verifiedAt = parsed
		}
		_, err = s.calculateWithKey(ctx, CalculateRewardInput{
			UserID:                  payload.UserID,
			SubmissionID:            payload.SubmissionID,
			CampaignID:              payload.CampaignID,
			VerificationCompletedAt: verifiedAt,
			EventID:                 event.EventID,
		}, "event:"+event.EventID)
	case domain.EventSubmissionViewLocked:
		var payload contracts.SubmissionViewLockedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode submission.view_locked payload: %w", err)
		}
		lockedAt := now
		if parsed, parseErr := time.Parse(time.RFC3339, payload.LockedAt); parseErr == nil {
			lockedAt = parsed
		}
		if err := s.snapshots.Upsert(ctx, ports.SubmissionViewSnapshot{
			SubmissionID: payload.SubmissionID,
			Views:        payload.LockedViews,
			PolledAt:     lockedAt,
		}); err != nil {
			return err
		}
		_, err = s.calculateWithKey(ctx, CalculateRewardInput{
			UserID:                  payload.UserID,
			SubmissionID:            payload.SubmissionID,
			CampaignID:              payload.CampaignID,
			LockedViews:             payload.LockedViews,
			VerificationCompletedAt: lockedAt,
			EventID:                 event.EventID,
		}, "event:"+event.EventID)
	case domain.EventSubmissionCancelled:
		var payload contracts.SubmissionCancelledPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode submission.cancelled payload: %w", err)
		}
		err = s.markCancelled(ctx, payload, event.EventID)
	case domain.EventTrackingMetricsUpdated:
		var payload contracts.TrackingMetricsUpdatedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode tracking.metrics.updated payload: %w", err)
		}
		polledAt := now
		if parsed, parseErr := time.Parse(time.RFC3339, payload.PolledAt); parseErr == nil {
			polledAt = parsed
		}
		err = s.snapshots.Upsert(ctx, ports.SubmissionViewSnapshot{
			SubmissionID: payload.TrackedPostID,
			Views:        payload.Views,
			PolledAt:     polledAt,
		})
	default:
		err = domain.ErrUnsupportedEventType
	}
	if err != nil {
		return err
	}

	return s.eventDedup.MarkProcessed(ctx, event.EventID, event.EventType, now.Add(s.cfg.EventDedupTTL))
}

func (s *Service) markCancelled(ctx context.Context, payload contracts.SubmissionCancelledPayload, eventID string) error {
	now := s.nowFn()
	reward := domain.Reward{
		SubmissionID:            payload.SubmissionID,
		UserID:                  payload.UserID,
		CampaignID:              payload.CampaignID,
		LockedViews:             0,
		RatePer1K:               0,
		GrossAmount:             0,
		NetAmount:               0,
		RolloverApplied:         0,
		FraudScore:              0,
		Status:                  domain.RewardStatusCancelled,
		VerificationCompletedAt: now,
		CalculatedAt:            now,
		LastEventID:             eventID,
	}
	if err := s.rewards.Save(ctx, reward); err != nil {
		return err
	}
	return s.audit.Append(ctx, ports.AuditRecord{
		LogID:        uuid.NewString(),
		SubmissionID: payload.SubmissionID,
		UserID:       payload.UserID,
		Action:       "submission_cancelled",
		Amount:       0,
		CreatedAt:    now,
		Metadata: map[string]string{
			"reason": payload.Reason,
		},
	})
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	pending, err := s.outbox.ListPending(ctx, s.cfg.OutboxFlushBatchSize)
	if err != nil {
		return err
	}
	for _, record := range pending {
		if record.EventClass != domain.CanonicalEventClassDomain {
			continue
		}
		if err := s.domainEvents.PublishDomain(ctx, record.Envelope); err != nil {
			return err
		}
		if err := s.outbox.MarkSent(ctx, record.RecordID, s.nowFn()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enqueueDomainRewardCalculated(ctx context.Context, reward domain.Reward) error {
	payload := contracts.RewardCalculatedPayload{
		SubmissionID:            reward.SubmissionID,
		UserID:                  reward.UserID,
		CampaignID:              reward.CampaignID,
		LockedViews:             reward.LockedViews,
		RatePer1K:               reward.RatePer1K,
		GrossAmount:             reward.GrossAmount,
		NetAmount:               reward.NetAmount,
		RolloverApplied:         reward.RolloverApplied,
		RolloverBalance:         reward.RolloverBalance,
		VerificationCompletedAt: reward.VerificationCompletedAt.Format(time.RFC3339),
		CalculatedAt:            reward.CalculatedAt.Format(time.RFC3339),
		Status:                  string(reward.Status),
		FraudScore:              reward.FraudScore,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{
		RecordID:   uuid.NewString(),
		EventClass: domain.CanonicalEventClassDomain,
		Envelope: contracts.EventEnvelope{
			EventID:          uuid.NewString(),
			EventType:        domain.EventRewardCalculated,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       reward.CalculatedAt,
			PartitionKeyPath: "data.submission_id",
			PartitionKey:     reward.SubmissionID,
			SourceService:    s.cfg.ServiceName,
			TraceID:          uuid.NewString(),
			SchemaVersion:    "v1",
			Data:             data,
		},
		CreatedAt: s.nowFn(),
	})
}

func (s *Service) enqueueDomainRewardPayoutEligible(ctx context.Context, reward domain.Reward) error {
	if !s.cfg.EnablePayoutEligibleEmission {
		return nil
	}
	eligibleAt := s.nowFn()
	if reward.EligibleAt != nil {
		eligibleAt = *reward.EligibleAt
	}
	payload := contracts.RewardPayoutEligiblePayload{
		SubmissionID:            reward.SubmissionID,
		UserID:                  reward.UserID,
		CampaignID:              reward.CampaignID,
		LockedViews:             reward.LockedViews,
		RatePer1K:               reward.RatePer1K,
		GrossAmount:             reward.GrossAmount,
		NetAmount:               reward.NetAmount,
		RolloverApplied:         reward.RolloverApplied,
		RolloverBalance:         reward.RolloverBalance,
		EligibleAt:              eligibleAt.Format(time.RFC3339),
		VerificationCompletedAt: reward.VerificationCompletedAt.Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{
		RecordID:   uuid.NewString(),
		EventClass: domain.CanonicalEventClassDomain,
		Envelope: contracts.EventEnvelope{
			EventID:          uuid.NewString(),
			EventType:        domain.EventRewardPayoutEligible,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       eligibleAt,
			PartitionKeyPath: "data.submission_id",
			PartitionKey:     reward.SubmissionID,
			SourceService:    s.cfg.ServiceName,
			TraceID:          uuid.NewString(),
			SchemaVersion:    "v1",
			Data:             data,
		},
		CreatedAt: s.nowFn(),
	})
}

func isSupportedEventType(eventType string) bool {
	switch eventType {
	case domain.EventSubmissionAutoApproved,
		domain.EventSubmissionCancelled,
		domain.EventSubmissionVerified,
		domain.EventSubmissionViewLocked,
		domain.EventTrackingMetricsUpdated:
		return true
	default:
		return false
	}
}

func validateDomainEventEnvelope(event contracts.EventEnvelope, allowedPartitionPaths ...string) error {
	if len(allowedPartitionPaths) == 0 {
		return fmt.Errorf("%w: missing partition key policy", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(event.EventID) == "" {
		return fmt.Errorf("%w: missing event_id", domain.ErrInvalidInput)
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("%w: missing occurred_at", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(event.SourceService) == "" {
		return fmt.Errorf("%w: missing source_service", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(event.TraceID) == "" {
		return fmt.Errorf("%w: missing trace_id", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(event.SchemaVersion) == "" {
		return fmt.Errorf("%w: missing schema_version", domain.ErrInvalidInput)
	}
	if len(event.Data) == 0 {
		return fmt.Errorf("%w: missing data payload", domain.ErrInvalidInput)
	}

	allowed := false
	for _, path := range allowedPartitionPaths {
		if event.PartitionKeyPath == path {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("%w: expected partition_key_path %s", domain.ErrInvalidInput, allowedPartitionPaths[0])
	}
	field := strings.TrimPrefix(event.PartitionKeyPath, "data.")
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("%w: invalid partition_key_path", domain.ErrInvalidInput)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("%w: invalid data payload", domain.ErrInvalidInput)
	}
	value, ok := payload[field]
	if !ok {
		return fmt.Errorf("%w: partition key field %s missing from payload", domain.ErrInvalidInput, field)
	}
	if fmt.Sprint(value) != event.PartitionKey {
		return fmt.Errorf("%w: partition key invariant failed", domain.ErrInvalidInput)
	}
	return nil
}
