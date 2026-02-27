package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/domain"
)

type RecommendationsRepository interface {
	SaveBatch(ctx context.Context, batch domain.RecommendationBatch) error
	GetLatestBatch(ctx context.Context, userID, role string) (domain.RecommendationBatch, error)
}

type FeedbackRepository interface {
	Create(ctx context.Context, row domain.FeedbackRecord) error
	ListByUser(ctx context.Context, userID string, limit int) ([]domain.FeedbackRecord, error)
}

type OverridesRepository interface {
	Upsert(ctx context.Context, row domain.RecommendationOverride) error
	ListActive(ctx context.Context, role string, now time.Time) ([]domain.RecommendationOverride, error)
	GetByID(ctx context.Context, overrideID string) (domain.RecommendationOverride, error)
}

type ModelsRepository interface {
	GetDefault(ctx context.Context) (domain.RecommendationModel, error)
	Upsert(ctx context.Context, model domain.RecommendationModel) error
}

type ABTestRepository interface {
	GetOrAssign(ctx context.Context, userID string, now time.Time) (domain.ABTestAssignment, error)
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
