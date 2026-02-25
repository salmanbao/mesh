package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"gorm.io/gorm"
)

type loginAttemptRepository struct {
	db *gorm.DB
}

func (r *loginAttemptRepository) Insert(ctx context.Context, attempt domain.LoginAttempt) error {
	rec := loginAttemptModel{
		UserID:        attempt.UserID,
		AttemptAt:     attempt.AttemptAt,
		IPAddress:     nullableString(attempt.IPAddress),
		Status:        attempt.Status,
		FailureReason: attempt.FailureReason,
		DeviceName:    attempt.DeviceName,
		DeviceOS:      attempt.DeviceOS,
		UserAgent:     attempt.UserAgent,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *loginAttemptRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, since *time.Time, status string) ([]domain.LoginAttempt, error) {
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID)
	if since != nil {
		query = query.Where("attempt_at >= ?", *since)
	}
	status = strings.TrimSpace(status)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var rows []loginAttemptModel
	if err := query.Order("attempt_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.LoginAttempt, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainLoginAttempt(row))
	}
	return result, nil
}
