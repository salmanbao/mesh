package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/domain"
)

type TrackedPostRepository interface {
	Create(ctx context.Context, row domain.TrackedPost) error
	GetByID(ctx context.Context, trackedPostID string) (domain.TrackedPost, error)
	Update(ctx context.Context, row domain.TrackedPost) error
	FindByUserPlatformURL(ctx context.Context, userID, platform, postURL string) (domain.TrackedPost, error)
	ListPollCandidates(ctx context.Context, before time.Time, limit int) ([]domain.TrackedPost, error)
}

type MetricSnapshotRepository interface {
	Append(ctx context.Context, row domain.MetricSnapshot) error
	ListByTrackedPostID(ctx context.Context, trackedPostID string) ([]domain.MetricSnapshot, error)
	LatestByTrackedPostID(ctx context.Context, trackedPostID string) (domain.MetricSnapshot, error)
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
