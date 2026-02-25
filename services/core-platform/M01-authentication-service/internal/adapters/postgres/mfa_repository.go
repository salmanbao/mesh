package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mfaRepository struct {
	db *gorm.DB
}

func (r *mfaRepository) ListEnabledMethods(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var methods []string
	if err := r.db.WithContext(ctx).
		Model(&twoFactorMethodModel{}).
		Where("user_id = ?", userID).
		Where("is_enabled = TRUE").
		Order("is_primary DESC, method_type ASC").
		Pluck("method_type", &methods).Error; err != nil {
		return nil, err
	}
	return methods, nil
}

func (r *mfaRepository) SetMethodEnabled(ctx context.Context, userID uuid.UUID, method string, enabled bool, isPrimary bool, updatedAt time.Time) error {
	rec := twoFactorMethodModel{
		UserID:     userID,
		MethodType: method,
		IsEnabled:  enabled,
		IsPrimary:  isPrimary,
		CreatedAt:  updatedAt,
		UpdatedAt:  updatedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "method_type"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"is_enabled": rec.IsEnabled,
			"is_primary": rec.IsPrimary,
			"updated_at": rec.UpdatedAt,
		}),
	}).Create(&rec).Error
}

func (r *mfaRepository) UpsertTOTPSecret(ctx context.Context, userID uuid.UUID, secretEncrypted []byte, updatedAt time.Time) error {
	rec := totpSecretModel{
		UserID:          userID,
		SecretEncrypted: secretEncrypted,
		CreatedAt:       updatedAt,
		ActivatedAt:     &updatedAt,
		DeactivatedAt:   nil,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"secret_encrypted": rec.SecretEncrypted,
			"activated_at":     rec.ActivatedAt,
			"deactivated_at":   nil,
		}),
	}).Create(&rec).Error
}

func (r *mfaRepository) ReplaceBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string, createdAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&backupCodeModel{}).Error; err != nil {
			return err
		}
		if len(codeHashes) == 0 {
			return nil
		}
		records := make([]backupCodeModel, 0, len(codeHashes))
		for _, hash := range codeHashes {
			records = append(records, backupCodeModel{
				UserID:    userID,
				CodeHash:  hash,
				CreatedAt: createdAt,
			})
		}
		return tx.Create(&records).Error
	})
}

func (r *mfaRepository) ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash string, usedAt time.Time) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&backupCodeModel{}).
		Where("user_id = ?", userID).
		Where("code_hash = ?", codeHash).
		Where("used_at IS NULL").
		Update("used_at", usedAt)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
