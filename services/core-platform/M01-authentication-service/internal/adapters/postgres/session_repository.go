package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

func (r *sessionRepository) Create(ctx context.Context, params ports.SessionCreateParams) (domain.Session, error) {
	rec := sessionModel{
		UserID:         params.UserID,
		DeviceName:     params.DeviceName,
		DeviceOS:       params.DeviceOS,
		IPAddress:      nullableString(params.IPAddress),
		UserAgent:      params.UserAgent,
		CreatedAt:      params.LastActivityAt,
		LastActivityAt: params.LastActivityAt,
		ExpiresAt:      params.ExpiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return domain.Session{}, err
	}
	return toDomainSession(rec), nil
}

func (r *sessionRepository) GetByID(ctx context.Context, sessionID uuid.UUID) (domain.Session, error) {
	var rec sessionModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Session{}, domain.ErrNotFound
		}
		return domain.Session{}, err
	}
	return toDomainSession(rec), nil
}

func (r *sessionRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Session, error) {
	var rows []sessionModel
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Session, 0, len(rows))
	for _, item := range rows {
		result = append(result, toDomainSession(item))
	}
	return result, nil
}

func (r *sessionRepository) TouchActivity(ctx context.Context, sessionID uuid.UUID, touchedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("session_id = ?", sessionID).
		Update("last_activity_at", touchedAt).Error
}

func (r *sessionRepository) RevokeByID(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("session_id = ?", sessionID).
		Where("revoked_at IS NULL").
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var exists int64
		if err := r.db.WithContext(ctx).Model(&sessionModel{}).Where("session_id = ?", sessionID).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return domain.ErrNotFound
		}
	}
	return nil
}

func (r *sessionRepository) RevokeAllByUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Update("revoked_at", revokedAt).Error
}
