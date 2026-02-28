package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/domain"
)

type ExportRequestRepository interface {
	Create(ctx context.Context, row domain.ExportRequest) error
	Update(ctx context.Context, row domain.ExportRequest) error
	GetByID(ctx context.Context, requestID string) (domain.ExportRequest, error)
	ListByUserID(ctx context.Context, userID string, limit int) ([]domain.ExportRequest, error)
}

type AuditRepository interface {
	Append(ctx context.Context, row domain.AuditLog) error
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}
