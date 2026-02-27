package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
)

type AssetRepository interface {
	Create(ctx context.Context, asset domain.MediaAsset) error
	GetByID(ctx context.Context, assetID string) (domain.MediaAsset, error)
	GetBySubmissionAndChecksum(ctx context.Context, submissionID, checksum string) (domain.MediaAsset, error)
	Update(ctx context.Context, asset domain.MediaAsset) error
}

type JobRepository interface {
	CreateMany(ctx context.Context, jobs []domain.MediaJob) error
	GetByID(ctx context.Context, jobID string) (domain.MediaJob, error)
	Update(ctx context.Context, job domain.MediaJob) error
	ListByAsset(ctx context.Context, assetID string) ([]domain.MediaJob, error)
	ListFailedByAsset(ctx context.Context, assetID string) ([]domain.MediaJob, error)
}

type OutputRepository interface {
	Upsert(ctx context.Context, output domain.MediaOutput) error
	ListByAsset(ctx context.Context, assetID string) ([]domain.MediaOutput, error)
}

type ThumbnailRepository interface {
	Upsert(ctx context.Context, thumbnail domain.MediaThumbnail) error
	ListByAsset(ctx context.Context, assetID string) ([]domain.MediaThumbnail, error)
}

type WatermarkRepository interface {
	Upsert(ctx context.Context, record domain.WatermarkRecord) error
	GetByAsset(ctx context.Context, assetID string) (domain.WatermarkRecord, error)
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
	Release(ctx context.Context, key string) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}
