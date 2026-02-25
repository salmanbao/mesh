package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type outboxRepository struct {
	db *gorm.DB
}

func (r *outboxRepository) Enqueue(ctx context.Context, event ports.OutboxEvent) error {
	rec := profileOutboxModel{
		OutboxID:         event.EventID,
		EventType:        event.EventType,
		PartitionKey:     event.PartitionKey,
		PartitionKeyPath: event.PartitionKeyPath,
		Payload:          string(event.Payload),
		SchemaVersion:    event.SchemaVersion,
		TraceID:          event.TraceID,
		CreatedAt:        event.OccurredAt,
		FirstSeenAt:      event.OccurredAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *outboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]ports.OutboxRecord, error) {
	var rows []profileOutboxModel
	if err := r.db.WithContext(ctx).Where("published_at IS NULL").Order("created_at asc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ports.OutboxRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.OutboxRecord{
			OutboxID: row.OutboxID, EventType: row.EventType, PartitionKey: row.PartitionKey,
			Payload: []byte(row.Payload), RetryCount: row.RetryCount, PublishedAt: row.PublishedAt,
			LastError: row.LastError, LastErrorAt: row.LastErrorAt, FirstSeenAt: row.FirstSeenAt,
		})
	}
	return out, nil
}

func (r *outboxRepository) MarkPublished(ctx context.Context, outboxID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&profileOutboxModel{}).Where("outbox_id = ?", outboxID).Update("published_at", at).Error
}

func (r *outboxRepository) MarkFailed(ctx context.Context, outboxID uuid.UUID, errMsg string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&profileOutboxModel{}).Where("outbox_id = ?", outboxID).Updates(map[string]any{
		"retry_count":   gorm.Expr("retry_count + 1"),
		"last_error":    errMsg,
		"last_error_at": at,
	}).Error
}
