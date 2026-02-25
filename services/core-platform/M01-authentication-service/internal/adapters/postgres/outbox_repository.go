package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type outboxRepository struct {
	db *gorm.DB
}

func (r *outboxRepository) Enqueue(ctx context.Context, event ports.OutboxEvent) error {
	rec := authOutboxModel{
		OutboxID:     event.EventID,
		EventType:    event.EventType,
		PartitionKey: event.PartitionKey,
		Payload:      string(event.Payload),
		CreatedAt:    event.OccurredAt,
		FirstSeenAt:  event.OccurredAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *outboxRepository) ClaimUnpublished(ctx context.Context, limit int, claimToken string, claimUntil time.Time) ([]ports.OutboxRecord, error) {
	if limit <= 0 {
		return nil, nil
	}
	if claimToken == "" {
		return nil, fmt.Errorf("claim token is required")
	}

	now := time.Now().UTC()
	var rows []authOutboxModel
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		subquery := tx.Model(&authOutboxModel{}).
			Select("outbox_id").
			Where("published_at IS NULL").
			Where("dead_lettered_at IS NULL").
			Where("claim_until IS NULL OR claim_until < ?", now).
			Order("created_at ASC").
			Limit(limit).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})

		if err := tx.Model(&authOutboxModel{}).
			Where("outbox_id IN (?)", subquery).
			Updates(map[string]any{
				"claim_token": claimToken,
				"claim_until": claimUntil,
			}).Error; err != nil {
			return err
		}

		return tx.Where("claim_token = ?", claimToken).
			Where("published_at IS NULL").
			Where("dead_lettered_at IS NULL").
			Order("created_at ASC").
			Find(&rows).Error
	}); err != nil {
		return nil, err
	}

	result := make([]ports.OutboxRecord, 0, len(rows))
	for _, row := range rows {
		item := ports.OutboxRecord{
			OutboxID:       row.OutboxID,
			EventType:      row.EventType,
			PartitionKey:   row.PartitionKey,
			Payload:        []byte(row.Payload),
			RetryCount:     row.RetryCount,
			LastError:      row.LastError,
			CreatedAt:      row.CreatedAt,
			PublishedAt:    row.PublishedAt,
			LastErrorAt:    row.LastErrorAt,
			FirstSeenAt:    row.FirstSeenAt,
			ClaimToken:     row.ClaimToken,
			ClaimUntil:     row.ClaimUntil,
			DeadLetteredAt: row.DeadLetteredAt,
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *outboxRepository) MarkPublished(ctx context.Context, outboxID uuid.UUID, claimToken string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authOutboxModel{}).
		Where("outbox_id = ?", outboxID).
		Where("claim_token = ?", claimToken).
		Updates(map[string]any{
			"published_at": at,
			"claim_token":  nil,
			"claim_until":  nil,
		}).Error
}

func (r *outboxRepository) MarkFailed(ctx context.Context, outboxID uuid.UUID, claimToken, errMsg string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authOutboxModel{}).
		Where("outbox_id = ?", outboxID).
		Where("claim_token = ?", claimToken).
		Updates(map[string]any{
			"retry_count":   gorm.Expr("retry_count + 1"),
			"last_error":    errMsg,
			"last_error_at": at,
			"claim_token":   nil,
			"claim_until":   nil,
		}).Error
}

func (r *outboxRepository) MarkDeadLettered(ctx context.Context, outboxID uuid.UUID, claimToken, errMsg string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authOutboxModel{}).
		Where("outbox_id = ?", outboxID).
		Where("claim_token = ?", claimToken).
		Updates(map[string]any{
			"retry_count":     gorm.Expr("retry_count + 1"),
			"last_error":      errMsg,
			"last_error_at":   at,
			"dead_lettered_at": at,
			"claim_token":     nil,
			"claim_until":     nil,
		}).Error
}
