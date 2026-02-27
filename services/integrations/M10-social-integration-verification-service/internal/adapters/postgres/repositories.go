package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/ports"
)

type Repositories struct {
	Accounts    *SocialAccountRepository
	Metrics     *SocialMetricRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Accounts:    &SocialAccountRepository{byID: map[string]domain.SocialAccount{}, byUserProvider: map[string]string{}},
		Metrics:     &SocialMetricRepository{rows: []domain.SocialMetric{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]eventDedupRow{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

func userProviderKey(userID, provider string) string { return strings.TrimSpace(userID) + ":" + strings.ToLower(strings.TrimSpace(provider)) }

type SocialAccountRepository struct {
	mu             sync.Mutex
	byID           map[string]domain.SocialAccount
	byUserProvider map[string]string
}

func (r *SocialAccountRepository) Create(_ context.Context, row domain.SocialAccount) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.byID[row.SocialAccountID]; ok { return domain.ErrConflict }
	k := userProviderKey(row.UserID, row.Provider)
	if _, ok := r.byUserProvider[k]; ok { return domain.ErrConflict }
	r.byID[row.SocialAccountID] = row
	r.byUserProvider[k] = row.SocialAccountID
	return nil
}
func (r *SocialAccountRepository) GetByID(_ context.Context, socialAccountID string) (domain.SocialAccount, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(socialAccountID)]
	if !ok { return domain.SocialAccount{}, domain.ErrNotFound }
	return row, nil
}
func (r *SocialAccountRepository) Update(_ context.Context, row domain.SocialAccount) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.byID[row.SocialAccountID]
	if !ok { return domain.ErrNotFound }
	delete(r.byUserProvider, userProviderKey(old.UserID, old.Provider))
	r.byID[row.SocialAccountID] = row
	r.byUserProvider[userProviderKey(row.UserID, row.Provider)] = row.SocialAccountID
	return nil
}
func (r *SocialAccountRepository) ListByUserID(_ context.Context, userID string) ([]domain.SocialAccount, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	uid := strings.TrimSpace(userID)
	out := make([]domain.SocialAccount, 0)
	for _, row := range r.byID { if row.UserID == uid { out = append(out, row) } }
	sort.Slice(out, func(i,j int) bool { return out[i].ConnectedAt.Before(out[j].ConnectedAt) })
	return out, nil
}
func (r *SocialAccountRepository) GetByUserProvider(_ context.Context, userID, provider string) (domain.SocialAccount, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	id, ok := r.byUserProvider[userProviderKey(userID, provider)]
	if !ok { return domain.SocialAccount{}, domain.ErrNotFound }
	row, ok := r.byID[id]
	if !ok { return domain.SocialAccount{}, domain.ErrNotFound }
	return row, nil
}

type SocialMetricRepository struct {
	mu   sync.Mutex
	rows []domain.SocialMetric
}

func (r *SocialMetricRepository) Append(_ context.Context, row domain.SocialMetric) error {
	r.mu.Lock(); defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *SocialMetricRepository) LatestByAccountID(_ context.Context, socialAccountID string) (domain.SocialMetric, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	for i := len(r.rows)-1; i >= 0; i-- {
		if r.rows[i].SocialAccountID == strings.TrimSpace(socialAccountID) { return r.rows[i], nil }
	}
	return domain.SocialMetric{}, domain.ErrNotFound
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}
func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok { return nil, nil }
	if now.After(row.ExpiresAt) { delete(r.rows, key); return nil, nil }
	cpy := row
	cpy.ResponseBody = append([]byte(nil), row.ResponseBody...)
	return &cpy, nil
}
func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if row, ok := r.rows[key]; ok && time.Now().UTC().Before(row.ExpiresAt) { return domain.ErrConflict }
	r.rows[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}
func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, _ time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok { return domain.ErrNotFound }
	row.ResponseCode = responseCode
	row.ResponseBody = append([]byte(nil), responseBody...)
	r.rows[key] = row
	return nil
}

type eventDedupRow struct {
	EventID string
	EventType string
	ExpiresAt time.Time
}

type EventDedupRepository struct {
	mu sync.Mutex
	rows map[string]eventDedupRow
}
func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	row, ok := r.rows[eventID]
	if !ok { return false, nil }
	if now.After(row.ExpiresAt) { delete(r.rows, eventID); return false, nil }
	return true, nil
}
func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock(); r.rows[eventID] = eventDedupRow{EventID: eventID, EventType: eventType, ExpiresAt: expiresAt}; return nil
}

type OutboxRepository struct {
	mu sync.Mutex
	rows map[string]ports.OutboxRecord
	order []string
}
func (r *OutboxRepository) Enqueue(_ context.Context, row ports.OutboxRecord) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.rows[row.RecordID]; ok { return domain.ErrConflict }
	r.rows[row.RecordID] = row
	r.order = append(r.order, row.RecordID)
	return nil
}
func (r *OutboxRepository) ListPending(_ context.Context, limit int) ([]ports.OutboxRecord, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	if limit <= 0 { limit = 100 }
	out := make([]ports.OutboxRecord, 0, limit)
	for _, id := range r.order {
		row, ok := r.rows[id]
		if !ok || row.SentAt != nil { continue }
		out = append(out, row)
		if len(out) >= limit { break }
	}
	return out, nil
}
func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock()
	row, ok := r.rows[recordID]
	if !ok { return domain.ErrNotFound }
	row.SentAt = &at
	r.rows[recordID] = row
	return nil
}
