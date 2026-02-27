package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/domain"
)

type ProductFileRepository interface {
	Create(ctx context.Context, row domain.ProductFile) error
	GetByProductID(ctx context.Context, productID string) (domain.ProductFile, error)
	Upsert(ctx context.Context, row domain.ProductFile) error
}

type DownloadTokenRepository interface {
	Create(ctx context.Context, row domain.DownloadToken) error
	GetByToken(ctx context.Context, token string) (domain.DownloadToken, error)
	FindActiveByUserProduct(ctx context.Context, userID, productID string, now time.Time) (domain.DownloadToken, error)
	Update(ctx context.Context, row domain.DownloadToken) error
	ListByProductUser(ctx context.Context, productID, userID string) ([]domain.DownloadToken, error)
}

type DownloadEventRepository interface {
	Append(ctx context.Context, row domain.DownloadEvent) error
	CountByIPSince(ctx context.Context, ip string, since time.Time) (int, error)
	LastByTokenUser(ctx context.Context, tokenID, userID string) (domain.DownloadEvent, error)
}

type DownloadRevocationAuditRepository interface {
	Append(ctx context.Context, row domain.DownloadRevocationAudit) error
	ListByProductUser(ctx context.Context, productID, userID string) ([]domain.DownloadRevocationAudit, error)
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
