package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"gorm.io/gorm"
)

type usernameHistoryRepository struct {
	db *gorm.DB
}

func (r *usernameHistoryRepository) ResolveRedirect(ctx context.Context, oldUsername string, now time.Time) (string, bool, error) {
	var rec usernameHistoryModel
	err := r.db.WithContext(ctx).
		Where("old_username = ? AND redirect_expires_at > ?", oldUsername, now).
		Order("changed_at desc").
		Take(&rec).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", false, nil
		}
		return "", false, err
	}
	return rec.NewUsername, true, nil
}

func (r *usernameHistoryRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]domain.UsernameHistory, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []usernameHistoryModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("changed_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UsernameHistory, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainUsernameHistory(row))
	}
	return out, nil
}
