package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/domain"
)

type EmbedSettingsRepository interface {
	GetByEntity(ctx context.Context, entityType, entityID string) (domain.EmbedSettings, error)
	Upsert(ctx context.Context, row domain.EmbedSettings) error
}

type EmbedCacheRepository interface {
	Get(ctx context.Context, cacheKey string, now time.Time) (domain.EmbedCache, error)
	Put(ctx context.Context, row domain.EmbedCache) error
	DeleteByEntity(ctx context.Context, entityType, entityID string) error
}

type ImpressionRepository interface {
	Append(ctx context.Context, row domain.Impression) error
	CountByIPSince(ctx context.Context, ipMasked string, since time.Time) (int, error)
	CountByEntityReferrerSince(ctx context.Context, entityType, entityID, referrerDomain string, since time.Time) (int, error)
	ListByEntityRange(ctx context.Context, entityType, entityID string, from, to *time.Time) ([]domain.Impression, error)
}

type InteractionRepository interface {
	Append(ctx context.Context, row domain.Interaction) error
	ListByEntityRange(ctx context.Context, entityType, entityID string, from, to *time.Time) ([]domain.Interaction, error)
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
