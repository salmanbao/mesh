package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type kycRepository struct {
	db *gorm.DB
}

func (r *kycRepository) CreateDocument(ctx context.Context, params ports.CreateKYCDocumentParams) (domain.KYCDocument, error) {
	rec := kycDocumentModel{
		UserID:       params.UserID,
		DocumentType: params.DocumentType,
		FileKey:      params.FileKey,
		Status:       params.Status,
		UploadedAt:   params.UploadedAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return domain.KYCDocument{}, err
	}
	return toDomainKYCDocument(rec), nil
}

func (r *kycRepository) ListDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.KYCDocument, error) {
	var rows []kycDocumentModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("uploaded_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.KYCDocument, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainKYCDocument(row))
	}
	return out, nil
}

func (r *kycRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, rejectionReason string, reviewedAt time.Time, reviewedBy *uuid.UUID) error {
	if err := r.db.WithContext(ctx).Model(&profileModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"kyc_status": status,
			"updated_at": reviewedAt,
		}).Error; err != nil {
		return err
	}

	update := map[string]any{
		"status":      string(status),
		"reviewed_at": reviewedAt,
		"reviewed_by": reviewedBy,
	}
	if rejectionReason != "" {
		update["rejection_reason"] = rejectionReason
	}
	return r.db.WithContext(ctx).Model(&kycDocumentModel{}).Where("user_id = ?", userID).Updates(update).Error
}

func (r *kycRepository) ListPendingQueue(ctx context.Context, limit, offset int) ([]domain.Profile, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []profileModel
	if err := r.db.WithContext(ctx).Where("kyc_status = ?", string(domain.KYCStatusPending)).
		Order("created_at asc").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Profile, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainProfile(row))
	}
	return out, nil
}
