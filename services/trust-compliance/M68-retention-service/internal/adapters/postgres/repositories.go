package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/ports"
)

type Repositories struct {
	Policies     *RetentionPolicyRepository
	Previews     *DeletionPreviewRepository
	Holds        *LegalHoldRepository
	Restorations *RestorationRepository
	Deletions    *ScheduledDeletionRepository
	Audit        *AuditRepository
	Idempotency  *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Policies:     &RetentionPolicyRepository{rowsByID: map[string]domain.RetentionPolicy{}},
		Previews:     &DeletionPreviewRepository{rowsByID: map[string]domain.DeletionPreview{}},
		Holds:        &LegalHoldRepository{rowsByID: map[string]domain.LegalHold{}},
		Restorations: &RestorationRepository{rowsByID: map[string]domain.RestorationRequest{}},
		Deletions:    &ScheduledDeletionRepository{rowsByID: map[string]domain.ScheduledDeletion{}},
		Audit:        &AuditRepository{rows: make([]domain.AuditLog, 0, 128)},
		Idempotency:  &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type RetentionPolicyRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.RetentionPolicy
}

func (r *RetentionPolicyRepository) Create(_ context.Context, row domain.RetentionPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.PolicyID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.PolicyID] = row
	return nil
}

func (r *RetentionPolicyRepository) GetByID(_ context.Context, policyID string) (domain.RetentionPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[policyID]
	if !ok {
		return domain.RetentionPolicy{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *RetentionPolicyRepository) List(_ context.Context) ([]domain.RetentionPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.RetentionPolicy, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return append([]domain.RetentionPolicy(nil), items...), nil
}

type DeletionPreviewRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.DeletionPreview
}

func (r *DeletionPreviewRepository) Create(_ context.Context, row domain.DeletionPreview) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.PreviewID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.PreviewID] = row
	return nil
}

func (r *DeletionPreviewRepository) GetByID(_ context.Context, previewID string) (domain.DeletionPreview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[previewID]
	if !ok {
		return domain.DeletionPreview{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *DeletionPreviewRepository) Update(_ context.Context, row domain.DeletionPreview) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.PreviewID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.PreviewID] = row
	return nil
}

type LegalHoldRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.LegalHold
}

func (r *LegalHoldRepository) Create(_ context.Context, row domain.LegalHold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.HoldID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.HoldID] = row
	return nil
}

func (r *LegalHoldRepository) List(_ context.Context, status string) ([]domain.LegalHold, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.LegalHold, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		if status == "" || row.Status == status {
			items = append(items, row)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return append([]domain.LegalHold(nil), items...), nil
}

type RestorationRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.RestorationRequest
}

func (r *RestorationRepository) Create(_ context.Context, row domain.RestorationRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.RestorationID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.RestorationID] = row
	return nil
}

func (r *RestorationRepository) GetByID(_ context.Context, restorationID string) (domain.RestorationRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[restorationID]
	if !ok {
		return domain.RestorationRequest{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *RestorationRepository) Update(_ context.Context, row domain.RestorationRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.RestorationID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.RestorationID] = row
	return nil
}

func (r *RestorationRepository) List(_ context.Context) ([]domain.RestorationRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.RestorationRequest, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return append([]domain.RestorationRequest(nil), items...), nil
}

type ScheduledDeletionRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.ScheduledDeletion
}

func (r *ScheduledDeletionRepository) Create(_ context.Context, row domain.ScheduledDeletion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.DeletionID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.DeletionID] = row
	return nil
}

func (r *ScheduledDeletionRepository) GetByPreviewID(_ context.Context, previewID string) (domain.ScheduledDeletion, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rowsByID {
		if row.PreviewID == previewID {
			return row, true, nil
		}
	}
	return domain.ScheduledDeletion{}, false, nil
}

func (r *ScheduledDeletionRepository) List(_ context.Context) ([]domain.ScheduledDeletion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.ScheduledDeletion, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ScheduledAt.Before(items[j].ScheduledAt) })
	return append([]domain.ScheduledDeletion(nil), items...), nil
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
