package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/domain"
)

type CacheRepository interface {
	Put(ctx context.Context, row domain.CacheEntry) error
	Get(ctx context.Context, key string, now time.Time) (domain.CacheItem, error)
	Delete(ctx context.Context, key string) (bool, error)
	Invalidate(ctx context.Context, keys []string) (int, error)
	MemoryUsedBytes(ctx context.Context) (int64, error)
}

type CacheMetricsRepository interface {
	RecordHit(ctx context.Context) error
	RecordMiss(ctx context.Context) error
	RecordEviction(ctx context.Context, count int) error
	SetMemoryUsed(ctx context.Context, bytes int64) error
	Snapshot(ctx context.Context) (domain.CacheMetrics, error)
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
