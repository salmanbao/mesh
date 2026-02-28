package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/domain"
)

type PlanRepository interface {
	Create(ctx context.Context, plan domain.MigrationPlan) error
	List(ctx context.Context) ([]domain.MigrationPlan, error)
	Get(ctx context.Context, planID string) (domain.MigrationPlan, error)
}

type RunRepository interface {
	Create(ctx context.Context, run domain.MigrationRun) error
	List(ctx context.Context) ([]domain.MigrationRun, error)
}

type RegistryRepository interface {
	Add(ctx context.Context, record domain.RegistryRecord) error
	List(ctx context.Context) ([]domain.RegistryRecord, error)
}

type BackfillRepository interface {
	Add(ctx context.Context, job domain.BackfillJob) error
	List(ctx context.Context) ([]domain.BackfillJob, error)
}

type MetricsRepository interface {
	Snapshot(ctx context.Context) (domain.Metrics, error)
	SetSnapshot(ctx context.Context, metrics domain.Metrics) error
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error)
	Upsert(ctx context.Context, rec domain.IdempotencyRecord) error
}
