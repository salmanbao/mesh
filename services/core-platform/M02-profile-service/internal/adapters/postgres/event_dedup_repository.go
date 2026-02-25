package postgres

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type eventDedupRepository struct {
	db *gorm.DB
}

func (r *eventDedupRepository) IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&profileEventDedupModel{}).
		Where("event_id = ? AND expires_at > ?", eventID, now).
		Count(&count).Error
	return count > 0, err
}

func (r *eventDedupRepository) MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error {
	rec := profileEventDedupModel{
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: time.Now().UTC(),
		ExpiresAt:   expiresAt,
	}
	return r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Assign(map[string]any{
			"event_type":   eventType,
			"processed_at": rec.ProcessedAt,
			"expires_at":   expiresAt,
		}).
		FirstOrCreate(&rec).Error
}

var _ ports.EventDedupRepository = (*eventDedupRepository)(nil)
