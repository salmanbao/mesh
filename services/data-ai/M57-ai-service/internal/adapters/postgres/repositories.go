package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/ports"
)

type Repositories struct {
	Predictions *PredictionRepository
	BatchJobs   *BatchJobRepository
	Models      *ModelRepository
	Feedback    *FeedbackRepository
	Audit       *AuditRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	return &Repositories{
		Predictions: &PredictionRepository{rowsByID: map[string]domain.Prediction{}},
		BatchJobs:   &BatchJobRepository{rowsByID: map[string]domain.BatchJob{}},
		Models: &ModelRepository{rowsByKey: map[string]domain.Model{
			modelKey("vf-core", "2026.02"): {
				ModelID:     "vf-core",
				Version:     "2026.02",
				DisplayName: "ViralForge Core Classifier",
				Active:      true,
				CreatedAt:   now,
			},
		}},
		Feedback:    &FeedbackRepository{rows: make([]domain.FeedbackLog, 0, 32)},
		Audit:       &AuditRepository{rows: make([]domain.AuditLog, 0, 64)},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type PredictionRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.Prediction
}

func (r *PredictionRepository) Create(_ context.Context, row domain.Prediction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.PredictionID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.PredictionID] = row
	return nil
}

func (r *PredictionRepository) GetByID(_ context.Context, predictionID string) (domain.Prediction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[predictionID]
	if !ok {
		return domain.Prediction{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *PredictionRepository) FindByKey(_ context.Context, contentHash, modelID, modelVersion string) (domain.Prediction, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rowsByID {
		if row.ContentHash == contentHash && row.ModelID == modelID && row.ModelVersion == modelVersion {
			return row, true, nil
		}
	}
	return domain.Prediction{}, false, nil
}

type BatchJobRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.BatchJob
}

func (r *BatchJobRepository) Create(_ context.Context, row domain.BatchJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.JobID]; ok {
		return domain.ErrConflict
	}
	cp := row
	cp.PredictionIDs = append([]string(nil), row.PredictionIDs...)
	cp.Predictions = append([]domain.Prediction(nil), row.Predictions...)
	r.rowsByID[row.JobID] = cp
	return nil
}

func (r *BatchJobRepository) GetByID(_ context.Context, jobID string) (domain.BatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[jobID]
	if !ok {
		return domain.BatchJob{}, domain.ErrNotFound
	}
	cp := row
	cp.PredictionIDs = append([]string(nil), row.PredictionIDs...)
	cp.Predictions = append([]domain.Prediction(nil), row.Predictions...)
	return cp, nil
}

type ModelRepository struct {
	mu        sync.Mutex
	rowsByKey map[string]domain.Model
}

func (r *ModelRepository) GetActive(_ context.Context, modelID, version string) (domain.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByKey[modelKey(modelID, version)]
	if !ok || !row.Active {
		return domain.Model{}, domain.ErrNotFound
	}
	return row, nil
}

func modelKey(modelID, version string) string {
	return modelID + "@" + version
}

type FeedbackRepository struct {
	mu   sync.Mutex
	rows []domain.FeedbackLog
}

func (r *FeedbackRepository) Append(_ context.Context, row domain.FeedbackLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}

type AuditRepository struct {
	mu   sync.Mutex
	rows []domain.AuditLog
}

func (r *AuditRepository) Append(_ context.Context, row domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok {
		return nil, nil
	}
	if !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	cp := row
	cp.ResponseBody = append([]byte(nil), row.ResponseBody...)
	return &cp, nil
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[key]; ok {
		if row.RequestHash != requestHash {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	r.rows[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[key]
	row.ResponseCode = responseCode
	row.ResponseBody = append([]byte(nil), responseBody...)
	if row.ExpiresAt.IsZero() {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.rows[key] = row
	return nil
}
