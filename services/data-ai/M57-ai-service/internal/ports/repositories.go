package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/domain"
)

type PredictionRepository interface {
	Create(ctx context.Context, row domain.Prediction) error
	GetByID(ctx context.Context, predictionID string) (domain.Prediction, error)
	FindByKey(ctx context.Context, contentHash, modelID, modelVersion string) (domain.Prediction, bool, error)
}

type BatchJobRepository interface {
	Create(ctx context.Context, row domain.BatchJob) error
	GetByID(ctx context.Context, jobID string) (domain.BatchJob, error)
}

type ModelRepository interface {
	GetActive(ctx context.Context, modelID, version string) (domain.Model, error)
}

type FeedbackRepository interface {
	Append(ctx context.Context, row domain.FeedbackLog) error
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
