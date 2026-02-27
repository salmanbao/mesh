package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/domain"
)

type SocialAccountRepository interface {
	Create(ctx context.Context, row domain.SocialAccount) error
	GetByID(ctx context.Context, socialAccountID string) (domain.SocialAccount, error)
	Update(ctx context.Context, row domain.SocialAccount) error
	ListByUserID(ctx context.Context, userID string) ([]domain.SocialAccount, error)
	GetByUserProvider(ctx context.Context, userID, provider string) (domain.SocialAccount, error)
}

type SocialMetricRepository interface {
	Append(ctx context.Context, row domain.SocialMetric) error
	LatestByAccountID(ctx context.Context, socialAccountID string) (domain.SocialMetric, error)
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

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
