package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/ports"
)

type Repositories struct {
	Idempotency *IdempotencyRepository
	Predictions *PredictionRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Idempotency: &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		Predictions: &PredictionRepository{records: map[string]domain.CampaignSuccessPrediction{}},
	}
}

type IdempotencyRepository struct {
	mu      sync.Mutex
	records map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.records, key)
		return nil, nil
	}
	clone := rec
	clone.ResponseBody = append([]byte(nil), rec.ResponseBody...)
	return &clone, nil
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.records[key]; ok && time.Now().UTC().Before(rec.ExpiresAt) {
		if rec.RequestHash != requestHash {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	r.records[key] = ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   expiresAt,
	}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return nil
	}
	rec.ResponseCode = responseCode
	rec.ResponseBody = append([]byte(nil), responseBody...)
	if at.After(rec.ExpiresAt) {
		rec.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.records[key] = rec
	return nil
}

type PredictionRepository struct {
	mu      sync.Mutex
	records map[string]domain.CampaignSuccessPrediction
}

func (r *PredictionRepository) SaveCampaignSuccess(_ context.Context, row domain.CampaignSuccessPrediction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.PredictionID] = row
	return nil
}
