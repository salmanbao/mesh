package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type profileRepository struct {
	db *gorm.DB
}

func (r *profileRepository) CreateProfileWithDefaults(ctx context.Context, params ports.CreateProfileParams) (domain.Profile, error) {
	rec := profileModel{
		UserID:      params.UserID,
		Username:    params.Username,
		DisplayName: params.DisplayName,
		Bio:         "",
		AvatarURL:   "",
		BannerURL:   "",
		KYCStatus:   string(domain.KYCStatusNotStarted),
		CreatedAt:   params.CreatedAt,
		UpdatedAt:   params.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.Profile{}, domain.ErrConflict
		}
		return domain.Profile{}, err
	}
	return toDomainProfile(rec), nil
}

func (r *profileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	var rec profileModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Profile{}, domain.ErrNotFound
		}
		return domain.Profile{}, err
	}
	return toDomainProfile(rec), nil
}

func (r *profileRepository) GetByUsername(ctx context.Context, username string) (domain.Profile, error) {
	var rec profileModel
	if err := r.db.WithContext(ctx).Where("username = ?", strings.ToLower(strings.TrimSpace(username))).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Profile{}, domain.ErrNotFound
		}
		return domain.Profile{}, err
	}
	return toDomainProfile(rec), nil
}

func (r *profileRepository) UpdateProfile(ctx context.Context, params ports.UpdateProfileParams) (domain.Profile, error) {
	updates := map[string]any{
		"updated_at": params.UpdatedAt,
	}
	if params.DisplayName != nil {
		updates["display_name"] = strings.TrimSpace(*params.DisplayName)
	}
	if params.Bio != nil {
		updates["bio"] = strings.TrimSpace(*params.Bio)
	}
	if params.IsPrivate != nil {
		updates["is_private"] = *params.IsPrivate
	}
	if params.IsUnlisted != nil {
		updates["is_unlisted"] = *params.IsUnlisted
	}
	if params.HideStatistics != nil {
		updates["hide_statistics"] = *params.HideStatistics
	}
	if params.AnalyticsOptOut != nil {
		updates["analytics_opt_out"] = *params.AnalyticsOptOut
	}
	if params.AvatarURL != nil {
		updates["avatar_url"] = *params.AvatarURL
	}
	if params.BannerURL != nil {
		updates["banner_url"] = *params.BannerURL
	}

	res := r.db.WithContext(ctx).Model(&profileModel{}).Where("user_id = ?", params.UserID).Updates(updates)
	if res.Error != nil {
		return domain.Profile{}, res.Error
	}
	if res.RowsAffected == 0 {
		return domain.Profile{}, domain.ErrNotFound
	}
	return r.GetByUserID(ctx, params.UserID)
}

func (r *profileRepository) UpdateUsername(ctx context.Context, userID uuid.UUID, newUsername string, now time.Time, redirectDays int) (string, domain.Profile, error) {
	var oldProfile profileModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Take(&oldProfile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		oldUsername := oldProfile.Username
		newUsername = strings.ToLower(strings.TrimSpace(newUsername))
		if oldUsername == newUsername {
			return nil
		}
		if err := tx.Model(&profileModel{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"username":                newUsername,
				"last_username_change_at": now,
				"updated_at":              now,
			}).Error; err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return err
		}
		if oldUsername != "" {
			history := usernameHistoryModel{
				UserID:            userID,
				OldUsername:       oldUsername,
				NewUsername:       newUsername,
				ChangedAt:         now,
				RedirectExpiresAt: now.Add(time.Duration(redirectDays) * 24 * time.Hour),
			}
			if err := tx.Create(&history).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", domain.Profile{}, err
	}
	updated, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return "", domain.Profile{}, err
	}
	return oldProfile.Username, updated, nil
}

func (r *profileRepository) CheckUsernameAvailability(ctx context.Context, username string) (ports.UsernameAvailability, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&profileModel{}).Where("username = ?", strings.ToLower(strings.TrimSpace(username))).Count(&count).Error; err != nil {
		return ports.UsernameAvailability{}, err
	}
	if count > 0 {
		return ports.UsernameAvailability{Available: false, Reason: "taken"}, nil
	}
	return ports.UsernameAvailability{Available: true}, nil
}

func (r *profileRepository) SoftDeleteByUserID(ctx context.Context, userID uuid.UUID, deletedAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&profileModel{}).Where("user_id = ?", userID).Updates(map[string]any{
		"deleted_at":   deletedAt,
		"updated_at":   deletedAt,
		"username":     "",
		"display_name": "deleted user",
		"bio":          "",
		"avatar_url":   "",
		"banner_url":   "",
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
