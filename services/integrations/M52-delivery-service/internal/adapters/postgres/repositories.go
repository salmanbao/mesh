package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/ports"
)

type Repositories struct {
	Files       *ProductFileRepository
	Tokens      *DownloadTokenRepository
	Downloads   *DownloadEventRepository
	Revocations *DownloadRevocationAuditRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Files:       &ProductFileRepository{byProduct: map[string]domain.ProductFile{}},
		Tokens:      &DownloadTokenRepository{byID: map[string]domain.DownloadToken{}, byToken: map[string]string{}},
		Downloads:   &DownloadEventRepository{rows: []domain.DownloadEvent{}},
		Revocations: &DownloadRevocationAuditRepository{rows: []domain.DownloadRevocationAudit{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
	}
}

type ProductFileRepository struct {
	mu        sync.Mutex
	byProduct map[string]domain.ProductFile
}

func (r *ProductFileRepository) Create(ctx context.Context, row domain.ProductFile) error {
	return r.Upsert(ctx, row)
}
func (r *ProductFileRepository) GetByProductID(_ context.Context, productID string) (domain.ProductFile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byProduct[productID]
	if !ok {
		return domain.ProductFile{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ProductFileRepository) Upsert(_ context.Context, row domain.ProductFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.byProduct[row.ProductID]; ok {
		row.CreatedAt = existing.CreatedAt
	}
	r.byProduct[row.ProductID] = row
	return nil
}

type DownloadTokenRepository struct {
	mu      sync.Mutex
	byID    map[string]domain.DownloadToken
	byToken map[string]string
}

func (r *DownloadTokenRepository) Create(_ context.Context, row domain.DownloadToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.TokenID]; ok {
		return domain.ErrConflict
	}
	if _, ok := r.byToken[row.Token]; ok {
		return domain.ErrConflict
	}
	r.byID[row.TokenID] = row
	r.byToken[row.Token] = row.TokenID
	return nil
}
func (r *DownloadTokenRepository) GetByToken(_ context.Context, token string) (domain.DownloadToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byToken[token]
	if !ok {
		return domain.DownloadToken{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok {
		return domain.DownloadToken{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *DownloadTokenRepository) FindActiveByUserProduct(_ context.Context, userID, productID string, now time.Time) (domain.DownloadToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var best *domain.DownloadToken
	for _, row := range r.byID {
		if row.UserID != userID || row.ProductID != productID {
			continue
		}
		if row.Revoked || !row.ExpiresAt.After(now) || row.DownloadCount >= row.MaxDownloads {
			continue
		}
		if best == nil || row.CreatedAt.After(best.CreatedAt) {
			cp := row
			best = &cp
		}
	}
	if best == nil {
		return domain.DownloadToken{}, domain.ErrNotFound
	}
	return *best, nil
}
func (r *DownloadTokenRepository) Update(_ context.Context, row domain.DownloadToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.TokenID]; !ok {
		return domain.ErrNotFound
	}
	r.byID[row.TokenID] = row
	r.byToken[row.Token] = row.TokenID
	return nil
}
func (r *DownloadTokenRepository) ListByProductUser(_ context.Context, productID, userID string) ([]domain.DownloadToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.DownloadToken{}
	for _, row := range r.byID {
		if row.ProductID == productID && row.UserID == userID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

type DownloadEventRepository struct {
	mu   sync.Mutex
	rows []domain.DownloadEvent
}

func (r *DownloadEventRepository) Append(_ context.Context, row domain.DownloadEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *DownloadEventRepository) CountByIPSince(_ context.Context, ip string, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, row := range r.rows {
		if row.IPAddress == ip && !row.Timestamp.Before(since) {
			count++
		}
	}
	return count, nil
}
func (r *DownloadEventRepository) LastByTokenUser(_ context.Context, tokenID, userID string) (domain.DownloadEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var best *domain.DownloadEvent
	for i := range r.rows {
		row := r.rows[i]
		if row.TokenID != tokenID || row.UserID != userID {
			continue
		}
		if best == nil || row.Timestamp.After(best.Timestamp) {
			cp := row
			best = &cp
		}
	}
	if best == nil {
		return domain.DownloadEvent{}, domain.ErrNotFound
	}
	return *best, nil
}

type DownloadRevocationAuditRepository struct {
	mu   sync.Mutex
	rows []domain.DownloadRevocationAudit
}

func (r *DownloadRevocationAuditRepository) Append(_ context.Context, row domain.DownloadRevocationAudit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *DownloadRevocationAuditRepository) ListByProductUser(_ context.Context, productID, userID string) ([]domain.DownloadRevocationAudit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.DownloadRevocationAudit{}
	for _, row := range r.rows {
		if row.ProductID == productID && row.UserID == userID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RevokedAt.After(out[j].RevokedAt) })
	return out, nil
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

type EventDedupRepository struct {
	mu   sync.Mutex
	rows map[string]time.Time
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	exp, ok := r.rows[eventID]
	if !ok {
		return false, nil
	}
	if now.After(exp) {
		delete(r.rows, eventID)
		return false, nil
	}
	return true, nil
}
func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, _ string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[eventID] = expiresAt
	return nil
}
