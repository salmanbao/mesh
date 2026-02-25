package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"gorm.io/gorm"
)

type credentialRepository struct {
	db *gorm.DB
}

func (r *credentialRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, updatedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"updated_at":    updatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *credentialRepository) SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool, updatedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"email_verified": verified,
			"updated_at":     updatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *credentialRepository) HasPassword(ctx context.Context, userID uuid.UUID) (bool, error) {
	var rec struct {
		PasswordHash *string `gorm:"column:password_hash"`
	}
	if err := r.db.WithContext(ctx).
		Model(&userModel{}).
		Select("password_hash").
		Where("user_id = ?", userID).
		Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, domain.ErrNotFound
		}
		return false, err
	}
	return rec.PasswordHash != nil && strings.TrimSpace(*rec.PasswordHash) != "", nil
}
