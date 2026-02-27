package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/domain"
)

type LayoutRepository interface {
	GetCurrent(ctx context.Context, userID, deviceType string) (domain.DashboardLayout, error)
	Save(ctx context.Context, layout domain.DashboardLayout) error
}

type CustomViewRepository interface {
	Create(ctx context.Context, view domain.CustomView) error
	GetByID(ctx context.Context, userID, viewID string) (domain.CustomView, error)
	ListByUser(ctx context.Context, userID string) ([]domain.CustomView, error)
}

type UserPreferenceRepository interface {
	GetByUser(ctx context.Context, userID string) (domain.UserPreference, error)
	Upsert(ctx context.Context, preference domain.UserPreference) error
}

type CacheInvalidationRepository interface {
	Add(ctx context.Context, row domain.CacheInvalidation) error
	ListByUser(ctx context.Context, userID string, limit int) ([]domain.CacheInvalidation, error)
}

type CachedDashboard struct {
	CacheKey  string
	Dashboard domain.Dashboard
	ExpiresAt time.Time
	UpdatedAt time.Time
}

type DashboardCacheRepository interface {
	Get(ctx context.Context, cacheKey string, now time.Time) (*CachedDashboard, error)
	Upsert(ctx context.Context, item CachedDashboard) error
	InvalidateByUser(ctx context.Context, userID string) error
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

type OutboxRecord struct {
	RecordID   string
	EventClass string
	Envelope   contracts.EventEnvelope
	CreatedAt  time.Time
	SentAt     *time.Time
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
