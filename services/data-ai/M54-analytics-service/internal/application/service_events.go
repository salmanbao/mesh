package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/domain"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if !domain.IsCanonicalAnalyticsInputEvent(envelope.EventType) {
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
	duplicate, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, now)
	if err != nil {
		return err
	}
	if duplicate {
		return nil
	}

	if err := s.applyEventToWarehouse(ctx, envelope); err != nil {
		return err
	}
	return s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, now.Add(s.cfg.EventDedupTTL))
}

func (s *Service) applyEventToWarehouse(ctx context.Context, envelope contracts.EventEnvelope) error {
	switch envelope.EventType {
	case domain.EventSubmissionCreated, domain.EventSubmissionApproved:
		var payload contracts.SubmissionPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode submission payload", domain.ErrInvalidEnvelope)
		}
		status := payload.Status
		if status == "" {
			if envelope.EventType == domain.EventSubmissionApproved {
				status = "approved"
			} else {
				status = "created"
			}
		}
		return s.warehouse.AddSubmission(ctx, domain.FactSubmission{
			SubmissionID: payload.SubmissionID,
			CreatorID:    payload.UserID,
			CampaignID:   payload.CampaignID,
			Platform:     strings.ToLower(strings.TrimSpace(payload.Platform)),
			Status:       status,
			Views:        payload.Views,
			OccurredAt:   nonZeroTime(envelope.OccurredAt),
		})
	case domain.EventPayoutPaid:
		var payload contracts.PayoutPaidPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode payout payload", domain.ErrInvalidEnvelope)
		}
		if err := s.warehouse.AddPayout(ctx, domain.FactPayout{
			PayoutID:    payload.PayoutID,
			CreatorID:   payload.UserID,
			Amount:      payload.Amount,
			OccurredAt:  nonZeroTime(envelope.OccurredAt),
			SourceEvent: envelope.EventType,
		}); err != nil {
			return err
		}
		return s.warehouse.UpsertDailyEarnings(ctx, domain.DailyEarnings{
			DayDate:     nonZeroTime(envelope.OccurredAt).Format("2006-01-02"),
			CreatorID:   payload.UserID,
			Payouts:     payload.Amount,
			NetEarnings: payload.Amount,
			UpdatedAt:   s.nowFn(),
		})
	case domain.EventRewardCalculated:
		var payload contracts.RewardCalculatedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode reward payload", domain.ErrInvalidEnvelope)
		}
		return s.warehouse.UpsertDailyEarnings(ctx, domain.DailyEarnings{
			DayDate:       nonZeroTime(envelope.OccurredAt).Format("2006-01-02"),
			CreatorID:     payload.UserID,
			GrossEarnings: payload.GrossAmount,
			NetEarnings:   payload.NetAmount,
			UpdatedAt:     s.nowFn(),
		})
	case domain.EventCampaignLaunched:
		var payload contracts.CampaignLaunchedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode campaign payload", domain.ErrInvalidEnvelope)
		}
		return s.warehouse.UpsertCampaign(ctx, domain.DimCampaign{
			CampaignID: payload.CampaignID,
			BrandID:    payload.BrandID,
			Category:   payload.Category,
			RewardRate: payload.RewardRate,
			Budget:     payload.Budget,
			LaunchedAt: nonZeroTime(envelope.OccurredAt),
		})
	case domain.EventUserRegistered:
		var payload contracts.UserRegisteredPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode user payload", domain.ErrInvalidEnvelope)
		}
		now := nonZeroTime(envelope.OccurredAt)
		return s.warehouse.UpsertUser(ctx, domain.DimUser{
			UserID:           payload.UserID,
			Role:             payload.Role,
			Country:          payload.Country,
			ConsentAnalytics: true,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	case domain.EventTransactionSucceeded, domain.EventTransactionRefunded:
		var payload contracts.TransactionPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode transaction payload", domain.ErrInvalidEnvelope)
		}
		refunded := envelope.EventType == domain.EventTransactionRefunded
		if err := s.warehouse.AddTransaction(ctx, domain.FactTransaction{
			TransactionID: payload.TransactionID,
			UserID:        payload.UserID,
			Amount:        payload.Amount,
			Refunded:      refunded,
			OccurredAt:    nonZeroTime(envelope.OccurredAt),
		}); err != nil {
			return err
		}
		row := domain.DailyEarnings{
			DayDate:       nonZeroTime(envelope.OccurredAt).Format("2006-01-02"),
			CreatorID:     payload.UserID,
			GrossEarnings: payload.Amount,
			NetEarnings:   payload.Amount,
			UpdatedAt:     s.nowFn(),
		}
		if refunded {
			row.Refunds = payload.Amount
			row.NetEarnings = -payload.Amount
		}
		return s.warehouse.UpsertDailyEarnings(ctx, row)
	case domain.EventTrackingMetricsUpdated, domain.EventDiscoverItemClicked, domain.EventDeliveryDownloadDone:
		var payload contracts.ClickPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode click payload", domain.ErrInvalidEnvelope)
		}
		clickID := payload.ClickID
		if clickID == "" {
			clickID = uuid.NewString()
		}
		return s.warehouse.AddClick(ctx, domain.FactClick{
			ClickID:     clickID,
			UserID:      payload.UserID,
			Platform:    payload.Platform,
			ItemType:    payload.ItemType,
			SessionID:   payload.SessionID,
			OccurredAt:  nonZeroTime(envelope.OccurredAt),
			SourceEvent: envelope.EventType,
		})
	case domain.EventConsentUpdated:
		var payload contracts.ConsentPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return fmt.Errorf("%w: decode consent payload", domain.ErrInvalidEnvelope)
		}
		now := nonZeroTime(envelope.OccurredAt)
		return s.warehouse.UpsertUser(ctx, domain.DimUser{
			UserID:           payload.UserID,
			ConsentAnalytics: payload.Analytics,
			UpdatedAt:        now,
		})
	default:
		return domain.ErrUnsupportedEventType
	}
}

func (s *Service) publishDLQIdempotencyConflict(ctx context.Context, key, traceID string) error {
	if s.dlq == nil {
		return nil
	}
	now := time.Now().UTC()
	return s.dlq.PublishDLQ(ctx, contracts.DLQRecord{
		OriginalEvent: contracts.EventEnvelope{
			EventID:          uuid.NewString(),
			EventType:        "analytics.idempotency.conflict",
			EventClass:       domain.CanonicalEventClassOps,
			OccurredAt:       now,
			PartitionKeyPath: "envelope.source_service",
			PartitionKey:     "M54-Analytics-Service",
			SourceService:    "M54-Analytics-Service",
			TraceID:          traceID,
			SchemaVersion:    "1.0",
			Data:             []byte(`{"source":"api"}`),
		},
		ErrorSummary: "idempotency key reused with mismatched payload",
		RetryCount:   1,
		FirstSeenAt:  now,
		LastErrorAt:  now,
		SourceTopic:  "api",
		TraceID:      traceID,
	})
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.EventType) == "" {
		return domain.ErrInvalidEnvelope
	}
	if event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.TraceID) == "" {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func validatePartitionKeyInvariant(event contracts.EventEnvelope, expectedPath string) error {
	if event.PartitionKeyPath != expectedPath || expectedPath == "" {
		return domain.ErrInvalidEnvelope
	}
	field := strings.TrimPrefix(event.PartitionKeyPath, "data.")
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return domain.ErrInvalidEnvelope
	}
	value, ok := payload[field]
	if !ok || fmt.Sprint(value) != event.PartitionKey {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func nonZeroTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}
