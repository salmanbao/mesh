package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type recoveryRepository struct {
	db *gorm.DB
}

func (r *recoveryRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error {
	rec := passwordResetTokenModel{
		UserID:    userID,
		TokenHash: tokenHash,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *recoveryRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, usedAt time.Time) (uuid.UUID, error) {
	var rec passwordResetTokenModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", tokenHash).
			Where("used_at IS NULL").
			Where("expires_at > ?", usedAt).
			Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		return tx.Model(&passwordResetTokenModel{}).
			Where("token_id = ?", rec.TokenID).
			Update("used_at", usedAt).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return rec.UserID, nil
}

func (r *recoveryRepository) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error {
	rec := emailVerificationTokenModel{
		UserID:    userID,
		TokenHash: tokenHash,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *recoveryRepository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string, verifiedAt time.Time) (uuid.UUID, error) {
	var rec emailVerificationTokenModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", tokenHash).
			Where("verified_at IS NULL").
			Where("expires_at > ?", verifiedAt).
			Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		return tx.Model(&emailVerificationTokenModel{}).
			Where("token_id = ?", rec.TokenID).
			Update("verified_at", verifiedAt).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return rec.UserID, nil
}
