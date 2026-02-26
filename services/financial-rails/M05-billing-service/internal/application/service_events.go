package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
)

func (s *Service) HandleDomainEvent(ctx context.Context, event contracts.EventEnvelope) error {
	if event.EventClass != domain.CanonicalEventClassDomain {
		return domain.ErrUnsupportedEventClass
	}
	if err := validateDomainEventEnvelope(event); err != nil {
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
	case "payout.paid":
		var payload contracts.PayoutPaidPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode payout.paid payload: %w", err)
		}
		paidAt, parseErr := time.Parse(time.RFC3339, payload.PaidAt)
		if parseErr != nil {
			paidAt = now
		}
		if err := s.invoices.CreatePayoutReceipt(ctx, domain.PayoutReceipt{
			ReceiptID:    uuid.NewString(),
			PayoutID:     payload.PayoutID,
			CreatorID:    payload.CreatorID,
			GrossAmount:  payload.GrossAmount,
			PlatformFee:  payload.FeeAmount,
			NetPayout:    payload.NetAmount,
			Currency:     payload.Currency,
			PayoutDate:   paidAt,
			PayoutStatus: "paid",
			CreatedAt:    now,
		}); err != nil {
			return err
		}
	case "payout.failed":
		var payload contracts.PayoutFailedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode payout.failed payload: %w", err)
		}
		_ = payload
	default:
		return domain.ErrUnsupportedEventType
	}

	return s.eventDedup.MarkProcessed(ctx, event.EventID, event.EventType, now.Add(s.cfg.EventDedupTTL))
}

func validateDomainEventEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" {
		return fmt.Errorf("%w: missing event_id", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(event.EventType) == "" {
		return fmt.Errorf("%w: missing event_type", domain.ErrInvalidInput)
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
	if !strings.HasPrefix(event.PartitionKeyPath, "data.") {
		return fmt.Errorf("%w: partition_key_path must be data.* for domain events", domain.ErrInvalidInput)
	}
	field := strings.TrimPrefix(event.PartitionKeyPath, "data.")
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("%w: partition_key_path field missing", domain.ErrInvalidInput)
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
