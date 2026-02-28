package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/ports"
)

type Repositories struct {
	ExportRequests *ExportRequestRepository
	Audit          *AuditRepository
	Idempotency    *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		ExportRequests: &ExportRequestRepository{rowsByID: map[string]domain.ExportRequest{}},
		Audit:          &AuditRepository{rows: make([]domain.AuditLog, 0, 128)},
		Idempotency:    &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type ExportRequestRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.ExportRequest
}

func (r *ExportRequestRepository) Create(_ context.Context, row domain.ExportRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.RequestID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.RequestID] = row
	return nil
}

func (r *ExportRequestRepository) Update(_ context.Context, row domain.ExportRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.RequestID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.RequestID] = row
	return nil
}

func (r *ExportRequestRepository) GetByID(_ context.Context, requestID string) (domain.ExportRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[requestID]
	if !ok {
		return domain.ExportRequest{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *ExportRequestRepository) ListByUserID(_ context.Context, userID string, limit int) ([]domain.ExportRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.ExportRequest, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		if userID == "" || row.UserID == userID {
			items = append(items, row)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RequestedAt.After(items[j].RequestedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return append([]domain.ExportRequest(nil), items...), nil
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
