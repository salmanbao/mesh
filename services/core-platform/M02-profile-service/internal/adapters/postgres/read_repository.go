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

type readRepository struct {
	db *gorm.DB
}

func (r *readRepository) GetProfileReadModelByUserID(ctx context.Context, userID uuid.UUID) (ports.ProfileReadModel, error) {
	var p profileModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ProfileReadModel{}, domain.ErrNotFound
		}
		return ports.ProfileReadModel{}, err
	}
	return r.buildReadModel(ctx, p)
}

func (r *readRepository) GetPublicProfileByUsername(ctx context.Context, username string, now time.Time) (ports.ProfileReadModel, bool, error) {
	var p profileModel
	if err := r.db.WithContext(ctx).
		Where("username = ? AND (deleted_at IS NULL OR deleted_at > ?)", strings.ToLower(strings.TrimSpace(username)), now).
		Take(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ProfileReadModel{}, false, nil
		}
		return ports.ProfileReadModel{}, false, err
	}
	rm, err := r.buildReadModel(ctx, p)
	return rm, true, err
}

func (r *readRepository) buildReadModel(ctx context.Context, p profileModel) (ports.ProfileReadModel, error) {
	var socials []socialLinkModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", p.UserID).Order("added_at asc").Find(&socials).Error; err != nil {
		return ports.ProfileReadModel{}, err
	}
	var payouts []payoutMethodModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", p.UserID).Find(&payouts).Error; err != nil {
		return ports.ProfileReadModel{}, err
	}
	var docs []kycDocumentModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", p.UserID).Order("uploaded_at desc").Find(&docs).Error; err != nil {
		return ports.ProfileReadModel{}, err
	}
	var stats profileStatsModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", p.UserID).Take(&stats).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ProfileReadModel{}, err
		}
		stats = profileStatsModel{UserID: p.UserID}
	}

	out := ports.ProfileReadModel{
		Profile: toDomainProfile(p),
		Stats:   toDomainProfileStats(stats),
	}
	for _, item := range socials {
		out.SocialLinks = append(out.SocialLinks, toDomainSocialLink(item))
	}
	for _, item := range payouts {
		out.PayoutMethods = append(out.PayoutMethods, toDomainPayoutMethod(item))
	}
	for _, item := range docs {
		out.Documents = append(out.Documents, toDomainKYCDocument(item))
	}
	return out, nil
}
