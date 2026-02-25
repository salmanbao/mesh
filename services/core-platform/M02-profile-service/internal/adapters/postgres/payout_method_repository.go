package postgres

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type payoutMethodRepository struct {
	db *gorm.DB
}

func (r *payoutMethodRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.PayoutMethod, error) {
	var rows []payoutMethodModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("added_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PayoutMethod, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainPayoutMethod(row))
	}
	return out, nil
}

func (r *payoutMethodRepository) Upsert(ctx context.Context, params ports.PutPayoutMethodParams) (domain.PayoutMethod, error) {
	method := strings.ToLower(strings.TrimSpace(params.MethodType))
	rec := payoutMethodModel{
		UserID:              params.UserID,
		MethodType:          method,
		IdentifierEncrypted: params.IdentifierEncrypted,
		VerificationStatus:  params.VerificationStatus,
		AddedAt:             params.Now,
	}
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND method_type = ?", params.UserID, method).
		Assign(map[string]any{
			"identifier_encrypted": params.IdentifierEncrypted,
			"verification_status":  params.VerificationStatus,
		}).
		FirstOrCreate(&rec).Error
	if err != nil {
		return domain.PayoutMethod{}, err
	}
	var out payoutMethodModel
	if err := r.db.WithContext(ctx).Where("user_id = ? AND method_type = ?", params.UserID, method).Take(&out).Error; err != nil {
		return domain.PayoutMethod{}, err
	}
	return toDomainPayoutMethod(out), nil
}
