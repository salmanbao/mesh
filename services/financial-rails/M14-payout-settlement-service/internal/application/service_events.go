package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/ports"
)

func (s *Service) HandleDomainEvent(ctx context.Context, event contracts.EventEnvelope) error {
	if event.EventType != domain.EventRewardPayoutEligible {
		return domain.ErrUnsupportedEventType
	}
	if event.EventClass != "" && event.EventClass != domain.CanonicalEventClassDomain {
		return domain.ErrUnsupportedEventClass
	}
	if err := validateDomainEventEnvelope(event, domain.EventRewardPayoutEligible, "data.submission_id"); err != nil {
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

	var payload contracts.RewardPayoutEligiblePayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("decode reward.payout_eligible payload: %w", err)
	}

	scheduledAt := now
	if parsed, err := time.Parse(time.RFC3339, payload.EligibleAt); err == nil {
		scheduledAt = parsed
	}
	_, err = s.requestPayoutWithKey(ctx, Actor{
		SubjectID: payload.UserID,
		Role:      "system",
		RequestID: event.TraceID,
	}, RequestPayoutInput{
		UserID:       payload.UserID,
		SubmissionID: payload.SubmissionID,
		Amount:       payload.GrossAmount,
		Currency:     s.cfg.DefaultCurrency,
		Method:       domain.PayoutMethodStandard,
		ScheduledAt:  scheduledAt,
	}, "event:"+event.EventID)
	if err != nil {
		return err
	}
	return s.eventDedup.MarkProcessed(ctx, event.EventID, event.EventType, now.Add(s.cfg.EventDedupTTL))
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
		now := s.nowFn()
		if err := s.outbox.MarkSent(ctx, record.RecordID, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) publishAnalyticsProcessing(ctx context.Context, payout domain.Payout) error {
	at := s.nowFn()
	payload := contracts.PayoutProcessingPayload{
		PayoutID:     payout.PayoutID,
		UserID:       payout.UserID,
		Amount:       payout.Amount,
		Method:       string(payout.Method),
		ProcessingAt: at.Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	envelope := contracts.EventEnvelope{
		EventID:          uuid.NewString(),
		EventType:        domain.EventPayoutProcessing,
		EventClass:       domain.CanonicalEventClassAnalyticsOnly,
		OccurredAt:       at,
		PartitionKeyPath: "data.payout_id",
		PartitionKey:     payout.PayoutID,
		SourceService:    s.cfg.ServiceName,
		TraceID:          uuid.NewString(),
		SchemaVersion:    "v1",
		Data:             data,
	}
	return s.analytics.PublishAnalytics(ctx, envelope)
}

func (s *Service) enqueueDomainPayoutPaid(ctx context.Context, payout domain.Payout) error {
	paidAt := s.nowFn()
	if payout.PaidAt != nil {
		paidAt = *payout.PaidAt
	}
	payload := contracts.PayoutPaidPayload{
		PayoutID: payout.PayoutID,
		UserID:   payout.UserID,
		Amount:   payout.Amount,
		Method:   string(payout.Method),
		PaidAt:   paidAt.Format(time.RFC3339),
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
			EventType:        domain.EventPayoutPaid,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       paidAt,
			PartitionKeyPath: "data.payout_id",
			PartitionKey:     payout.PayoutID,
			SourceService:    s.cfg.ServiceName,
			TraceID:          uuid.NewString(),
			SchemaVersion:    "v1",
			Data:             data,
		},
		CreatedAt: s.nowFn(),
	})
}

func (s *Service) enqueueDomainPayoutFailed(ctx context.Context, payout domain.Payout) error {
	failedAt := s.nowFn()
	if payout.FailedAt != nil {
		failedAt = *payout.FailedAt
	}
	payload := contracts.PayoutFailedPayload{
		PayoutID: payout.PayoutID,
		UserID:   payout.UserID,
		Amount:   payout.Amount,
		Method:   string(payout.Method),
		FailedAt: failedAt.Format(time.RFC3339),
		Reason:   payout.FailureReason,
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
			EventType:        domain.EventPayoutFailed,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       failedAt,
			PartitionKeyPath: "data.payout_id",
			PartitionKey:     payout.PayoutID,
			SourceService:    s.cfg.ServiceName,
			TraceID:          uuid.NewString(),
			SchemaVersion:    "v1",
			Data:             data,
		},
		CreatedAt: s.nowFn(),
	})
}

func validateDomainEventEnvelope(event contracts.EventEnvelope, expectedEventType, expectedPartitionPath string) error {
	if strings.TrimSpace(event.EventID) == "" {
		return fmt.Errorf("%w: missing event_id", domain.ErrInvalidInput)
	}
	if event.EventType != expectedEventType {
		return fmt.Errorf("%w: unsupported event_type %s", domain.ErrInvalidInput, event.EventType)
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
	if event.PartitionKeyPath != expectedPartitionPath {
		return fmt.Errorf("%w: expected partition_key_path %s", domain.ErrInvalidInput, expectedPartitionPath)
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
