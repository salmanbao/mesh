package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/domain"
)

type Repositories struct {
	Plans       *PlanRepository
	Runs        *RunRepository
	Registry    *RegistryRepository
	Backfills   *BackfillRepository
	Metrics     *MetricsRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Plans:       &PlanRepository{rows: []domain.MigrationPlan{}},
		Runs:        &RunRepository{rows: []domain.MigrationRun{}},
		Registry:    &RegistryRepository{rows: []domain.RegistryRecord{}},
		Backfills:   &BackfillRepository{rows: []domain.BackfillJob{}},
		Metrics:     &MetricsRepository{snapshot: domain.Metrics{}},
		Idempotency: &IdempotencyRepository{rows: map[string]domain.IdempotencyRecord{}},
	}
}

type PlanRepository struct {
	mu   sync.Mutex
	rows []domain.MigrationPlan
}

func (r *PlanRepository) Create(_ context.Context, plan domain.MigrationPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.PlanID == plan.PlanID {
			return domain.ErrConflict
		}
	}
	r.rows = append(r.rows, plan)
	return nil
}

func (r *PlanRepository) List(_ context.Context) ([]domain.MigrationPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.MigrationPlan, len(r.rows))
	copy(out, r.rows)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *PlanRepository) Get(_ context.Context, planID string) (domain.MigrationPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.PlanID == strings.TrimSpace(planID) {
			return row, nil
		}
	}
	return domain.MigrationPlan{}, domain.ErrNotFound
}

type RunRepository struct {
	mu   sync.Mutex
	rows []domain.MigrationRun
}

func (r *RunRepository) Create(_ context.Context, run domain.MigrationRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, run)
	return nil
}

func (r *RunRepository) List(_ context.Context) ([]domain.MigrationRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.MigrationRun, len(r.rows))
	copy(out, r.rows)
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out, nil
}

type RegistryRepository struct {
	mu   sync.Mutex
	rows []domain.RegistryRecord
}

func (r *RegistryRepository) Add(_ context.Context, record domain.RegistryRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, record)
	return nil
}

func (r *RegistryRepository) List(_ context.Context) ([]domain.RegistryRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.RegistryRecord, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

type BackfillRepository struct {
	mu   sync.Mutex
	rows []domain.BackfillJob
}

func (r *BackfillRepository) Add(_ context.Context, job domain.BackfillJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, job)
	return nil
}

func (r *BackfillRepository) List(_ context.Context) ([]domain.BackfillJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.BackfillJob, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

type MetricsRepository struct {
	mu       sync.Mutex
	snapshot domain.Metrics
}

func (r *MetricsRepository) Snapshot(_ context.Context) (domain.Metrics, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot, nil
}

func (r *MetricsRepository) SetSnapshot(_ context.Context, metrics domain.Metrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = metrics
	return nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.rows[strings.TrimSpace(key)]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	copy := rec
	return &copy, nil
}

func (r *IdempotencyRepository) Upsert(_ context.Context, rec domain.IdempotencyRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[rec.Key] = rec
	return nil
}
