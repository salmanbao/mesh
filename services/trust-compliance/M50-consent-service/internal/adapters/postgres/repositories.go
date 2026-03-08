package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/ports"
)

type Repositories struct {
	Consents    *ConsentRepository
	Audit       *AuditRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Consents: &ConsentRepository{
			rowsByUserID:  map[string]domain.ConsentRecord{},
			historyByUser: map[string][]domain.ConsentHistory{},
		},
		Audit:       &AuditRepository{rows: make([]domain.AuditLog, 0, 128)},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type ConsentRepository struct {
	mu            sync.Mutex
	rowsByUserID  map[string]domain.ConsentRecord
	historyByUser map[string][]domain.ConsentHistory
}

func (r *ConsentRepository) Upsert(_ context.Context, row domain.ConsentRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rowsByUserID[row.UserID] = cloneRecord(row)
	return nil
}

func (r *ConsentRepository) GetByUserID(_ context.Context, userID string) (domain.ConsentRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByUserID[userID]
	if !ok {
		return domain.ConsentRecord{}, domain.ErrNotFound
	}
	return cloneRecord(row), nil
}

func (r *ConsentRepository) AppendHistory(_ context.Context, row domain.ConsentHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.historyByUser[row.UserID] = append(r.historyByUser[row.UserID], row)
	return nil
}

func (r *ConsentRepository) ListHistory(_ context.Context, userID string, limit int) ([]domain.ConsentHistory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := append([]domain.ConsentHistory(nil), r.historyByUser[userID]...)
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt.After(items[j].OccurredAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
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

func cloneRecord(row domain.ConsentRecord) domain.ConsentRecord {
	clone := row
	clone.Preferences = make(map[string]bool, len(row.Preferences))
	for key, value := range row.Preferences {
		clone.Preferences[key] = value
	}
	return clone
}
