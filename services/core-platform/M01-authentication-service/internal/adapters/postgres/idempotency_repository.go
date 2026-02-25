package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
)

type idempotencyRepository struct {
	db *gorm.DB
}

func (r *idempotencyRepository) Get(ctx context.Context, key string) (*ports.IdempotencyRecord, error) {
	var rec authIdempotencyModel
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := ports.IdempotencyRecord{
		Key:          rec.IdempotencyKey,
		RequestHash:  rec.RequestHash,
		Status:       rec.Status,
		ResponseCode: rec.ResponseCode,
		ExpiresAt:    rec.ExpiresAt,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
	if rec.ResponseBody != nil {
		out.ResponseBody = []byte(*rec.ResponseBody)
	}
	return &out, nil
}

func (r *idempotencyRepository) Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error {
	rec := authIdempotencyModel{
		IdempotencyKey: key,
		RequestHash:    requestHash,
		Status:         "PENDING",
		ExpiresAt:      expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (r *idempotencyRepository) Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	var body *string
	if len(responseBody) > 0 {
		raw := string(responseBody)
		body = &raw
	}
	return r.db.WithContext(ctx).
		Model(&authIdempotencyModel{}).
		Where("idempotency_key = ?", key).
		Updates(map[string]any{
			"status":        "COMPLETED",
			"response_code": responseCode,
			"response_body": body,
			"updated_at":    at,
		}).Error
}
