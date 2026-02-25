package postgres

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type socialLinkRepository struct {
	db *gorm.DB
}

func (r *socialLinkRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.SocialLink, error) {
	var rows []socialLinkModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("added_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SocialLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainSocialLink(row))
	}
	return out, nil
}

func (r *socialLinkRepository) Create(ctx context.Context, params ports.CreateSocialLinkParams) (domain.SocialLink, error) {
	rec := socialLinkModel{
		UserID:            params.UserID,
		Platform:          strings.ToLower(strings.TrimSpace(params.Platform)),
		Handle:            params.Handle,
		ProfileURL:        params.ProfileURL,
		Verified:          params.Verified,
		OAuthConnectionID: params.OAuthConnectionID,
		AddedAt:           params.AddedAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.SocialLink{}, domain.ErrConflict
		}
		return domain.SocialLink{}, err
	}
	return toDomainSocialLink(rec), nil
}

func (r *socialLinkRepository) DeleteByUserAndPlatform(ctx context.Context, userID uuid.UUID, platform string) error {
	res := r.db.WithContext(ctx).Where("user_id = ? AND platform = ?", userID, strings.ToLower(strings.TrimSpace(platform))).Delete(&socialLinkModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *socialLinkRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&socialLinkModel{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
