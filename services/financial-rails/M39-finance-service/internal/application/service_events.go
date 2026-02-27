package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
)

func (s *Service) HandleDomainEvent(_ context.Context, event contracts.EventEnvelope) error {
	if event.EventClass != "" && event.EventClass != domain.CanonicalEventClassDomain {
		return domain.ErrUnsupportedEventClass
	}
	return domain.ErrUnsupportedEventType
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	pending, err := s.outbox.ListPending(ctx, s.cfg.OutboxFlushBatchSize)
	if err != nil {
		return err
	}

	for _, record := range pending {
		now := s.nowFn()
		switch record.EventClass {
		case domain.CanonicalEventClassDomain:
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
			if err := s.domainEvents.PublishDomain(ctx, record.Envelope); err != nil {
				return err
			}
			if err := s.eventDedup.MarkProcessed(ctx, record.Envelope.EventID, record.Envelope.EventType, now.Add(s.cfg.EventDedupTTL)); err != nil {
				return err
			}
		case domain.CanonicalEventClassAnalyticsOnly:
			dup, err := s.eventDedup.IsDuplicate(ctx, record.Envelope.EventID, now)
			if err != nil {
				return err
			}
			if !dup {
				if err := s.analytics.PublishAnalytics(ctx, record.Envelope); err != nil {
					return err
				}
				if err := s.eventDedup.MarkProcessed(ctx, record.Envelope.EventID, record.Envelope.EventType, now.Add(s.cfg.EventDedupTTL)); err != nil {
					return err
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

func (s *Service) enqueueDomainTransactionSucceeded(ctx context.Context, transaction domain.Transaction) error {
	occurredAt := s.nowFn()
	if transaction.SucceededAt != nil {
		occurredAt = *transaction.SucceededAt
	}
	payload := contracts.TransactionSucceededPayload{
		TransactionID: transaction.TransactionID,
		UserID:        transaction.UserID,
		Amount:        transaction.Amount,
		Currency:      transaction.Currency,
		Provider:      string(transaction.Provider),
		OccurredAt:    occurredAt.Format(time.RFC3339),
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
			EventType:        domain.EventTransactionSucceeded,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       occurredAt,
			PartitionKeyPath: "data.transaction_id",
			PartitionKey:     transaction.TransactionID,
			SourceService:    s.cfg.ServiceName,
			TraceID:          uuid.NewString(),
			SchemaVersion:    "v1",
			Data:             data,
		},
		CreatedAt: s.nowFn(),
	})
}

func (s *Service) enqueueDomainTransactionFailed(ctx context.Context, transaction domain.Transaction) error {
	occurredAt := s.nowFn()
	if transaction.FailedAt != nil {
		occurredAt = *transaction.FailedAt
	}
	payload := contracts.TransactionFailedPayload{
		TransactionID: transaction.TransactionID,
		UserID:        transaction.UserID,
		Amount:        transaction.Amount,
		Currency:      transaction.Currency,
		Provider:      string(transaction.Provider),
		OccurredAt:    occurredAt.Format(time.RFC3339),
		Reason:        transaction.FailureReason,
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
			EventType:        domain.EventTransactionFailed,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       occurredAt,
			PartitionKeyPath: "data.transaction_id",
			PartitionKey:     transaction.TransactionID,
			SourceService:    s.cfg.ServiceName,
			TraceID:          uuid.NewString(),
			SchemaVersion:    "v1",
			Data:             data,
		},
		CreatedAt: s.nowFn(),
	})
}

func (s *Service) enqueueDomainTransactionRefunded(ctx context.Context, transaction domain.Transaction, refund domain.Refund) error {
	occurredAt := s.nowFn()
	if transaction.RefundedAt != nil {
		occurredAt = *transaction.RefundedAt
	}
	payload := contracts.TransactionRefundedPayload{
		TransactionID: transaction.TransactionID,
		RefundID:      refund.RefundID,
		UserID:        transaction.UserID,
		Amount:        refund.Amount,
		Currency:      refund.Currency,
		Provider:      string(transaction.Provider),
		OccurredAt:    occurredAt.Format(time.RFC3339),
		Reason:        refund.Reason,
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
			EventType:        domain.EventTransactionRefunded,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       occurredAt,
			PartitionKeyPath: "data.transaction_id",
			PartitionKey:     transaction.TransactionID,
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
