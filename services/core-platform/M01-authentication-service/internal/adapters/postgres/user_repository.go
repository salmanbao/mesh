package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func (r *userRepository) CreateWithOutboxTx(ctx context.Context, params ports.CreateUserTxParams, outboxEvent ports.OutboxEvent) (domain.User, error) {
	var result domain.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role roleModel
		if err := tx.Where("name = ?", params.RoleName).Take(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrRoleResolutionFailed
			}
			return err
		}

		rec := userModel{
			Email:         params.Email,
			PasswordHash:  params.PasswordHash,
			RoleID:        role.RoleID,
			EmailVerified: params.EmailVerified,
			CreatedAt:     params.RegisteredAtUTC,
			UpdatedAt:     params.RegisteredAtUTC,
		}
		if err := tx.Create(&rec).Error; err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return err
		}

		payload := outboxEvent.Payload
		if len(payload) == 0 {
			payload = []byte(`{}`)
		}
		var payloadObj map[string]any
		if err := json.Unmarshal(payload, &payloadObj); err == nil {
			payloadObj["user_id"] = rec.UserID.String()
			if adjusted, mErr := json.Marshal(payloadObj); mErr == nil {
				payload = adjusted
			}
		}

		outbox := authOutboxModel{
			OutboxID:     outboxEvent.EventID,
			EventType:    outboxEvent.EventType,
			PartitionKey: rec.UserID.String(),
			Payload:      string(payload),
			CreatedAt:    outboxEvent.OccurredAt,
			FirstSeenAt:  outboxEvent.OccurredAt,
		}
		if err := tx.Create(&outbox).Error; err != nil {
			return err
		}

		result = toDomainUser(rec, role.Name)
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}
	return result, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var rec userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	roleName, err := r.loadRoleName(ctx, rec.RoleID)
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(rec, roleName), nil
}

func (r *userRepository) GetByID(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	var rec userModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	roleName, err := r.loadRoleName(ctx, rec.RoleID)
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(rec, roleName), nil
}

func (r *userRepository) Deactivate(ctx context.Context, userID uuid.UUID, deactivatedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_active":  false,
			"deleted_at": deactivatedAt,
			"updated_at": deactivatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *userRepository) loadRoleName(ctx context.Context, roleID uuid.UUID) (string, error) {
	var role roleModel
	if err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Take(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ErrRoleResolutionFailed
		}
		return "", err
	}
	return role.Name, nil
}
