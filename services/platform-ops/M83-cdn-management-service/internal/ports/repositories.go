package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/domain"
)

type ConfigRepository interface {
	Create(ctx context.Context, config domain.CDNConfig) error
	List(ctx context.Context) ([]domain.CDNConfig, error)
	Latest(ctx context.Context) (domain.CDNConfig, error)
}

type PurgeRepository interface {
	Create(ctx context.Context, request domain.PurgeRequest) error
	List(ctx context.Context) ([]domain.PurgeRequest, error)
}

type MetricsRepository interface {
	Snapshot(ctx context.Context) (domain.Metrics, error)
	SetSnapshot(ctx context.Context, metrics domain.Metrics) error
}

type CertificateRepository interface {
	List(ctx context.Context) ([]domain.Certificate, error)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error)
	Upsert(ctx context.Context, rec domain.IdempotencyRecord) error
}
