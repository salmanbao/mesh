package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type profileStatsRepository struct {
	db *gorm.DB
}

func (r *profileStatsRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (domain.ProfileStats, error) {
	var rec profileStatsModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ProfileStats{}, domain.ErrNotFound
		}
		return domain.ProfileStats{}, err
	}
	return toDomainProfileStats(rec), nil
}

func (r *profileStatsRepository) Upsert(ctx context.Context, params ports.UpsertProfileStatsParams) error {
	rec := profileStatsModel{
		UserID:           params.UserID,
		TotalEarningsYTD: params.TotalEarningsYTD,
		SubmissionCount:  params.SubmissionCount,
		ApprovalRate:     params.ApprovalRate,
		FollowerCount:    params.FollowerCount,
		LastUpdatedAt:    params.UpdatedAt,
	}
	return r.db.WithContext(ctx).
		Where("user_id = ?", params.UserID).
		Assign(map[string]any{
			"total_earnings_ytd": params.TotalEarningsYTD,
			"submission_count":   params.SubmissionCount,
			"approval_rate":      params.ApprovalRate,
			"follower_count":     params.FollowerCount,
			"last_updated_at":    params.UpdatedAt,
		}).
		FirstOrCreate(&rec).Error
}
