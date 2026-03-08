package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/domain"
)

type ConsentRepository interface {
	Upsert(ctx context.Context, row domain.ConsentRecord) error
	GetByUserID(ctx context.Context, userID string) (domain.ConsentRecord, error)
	AppendHistory(ctx context.Context, row domain.ConsentHistory) error
	ListHistory(ctx context.Context, userID string, limit int) ([]domain.ConsentHistory, error)
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
