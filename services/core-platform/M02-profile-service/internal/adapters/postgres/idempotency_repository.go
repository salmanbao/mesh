package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type idempotencyRepository struct {
	db *gorm.DB
}

func (r *idempotencyRepository) Get(ctx context.Context, key string) (*ports.IdempotencyRecord, error) {
	var rec profileIdempotencyModel
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := &ports.IdempotencyRecord{
		Key: rec.IdempotencyKey, RequestHash: rec.RequestHash, Status: rec.Status,
		ResponseCode: rec.ResponseCode, ExpiresAt: rec.ExpiresAt,
	}
	if rec.ResponseBody != nil {
		out.ResponseBody = []byte(*rec.ResponseBody)
	}
	return out, nil
}

func (r *idempotencyRepository) Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error {
	rec := profileIdempotencyModel{
		IdempotencyKey: key,
		RequestHash:    requestHash,
		Status:         "reserved",
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	err := r.db.WithContext(ctx).Create(&rec).Error
	if err != nil {
		if isUniqueViolation(err) {
			return errors.New("already reserved")
		}
		return err
	}
	return nil
}

func (r *idempotencyRepository) Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	payload := string(responseBody)
	return r.db.WithContext(ctx).Model(&profileIdempotencyModel{}).
		Where("idempotency_key = ?", key).
		Updates(map[string]any{
			"status":        "completed",
			"response_code": responseCode,
			"response_body": payload,
			"updated_at":    at,
		}).Error
}
