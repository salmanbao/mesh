package postgres

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/ports"
)

type Repositories struct {
	Payouts     *PayoutRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Payouts: &PayoutRepository{
			payouts: make(map[string]domain.Payout),
		},
		Idempotency: &IdempotencyRepository{
			records: make(map[string]ports.IdempotencyRecord),
		},
		EventDedup: &EventDedupRepository{
			records: make(map[string]dedupRecord),
		},
		Outbox: &OutboxRepository{
			records: make(map[string]ports.OutboxRecord),
		},
	}
}

type PayoutRepository struct {
	mu      sync.RWMutex
	payouts map[string]domain.Payout
	order   []string
}

func (r *PayoutRepository) Create(_ context.Context, payout domain.Payout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payouts[payout.PayoutID] = payout
	r.order = append(r.order, payout.PayoutID)
	return nil
}

func (r *PayoutRepository) Update(_ context.Context, payout domain.Payout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.payouts[payout.PayoutID]; !ok {
		return domain.ErrNotFound
	}
	r.payouts[payout.PayoutID] = payout
	return nil
}

func (r *PayoutRepository) GetByID(_ context.Context, payoutID string) (domain.Payout, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	payout, ok := r.payouts[payoutID]
	if !ok {
		return domain.Payout{}, domain.ErrNotFound
	}
	return payout, nil
}

func (r *PayoutRepository) List(_ context.Context, query ports.HistoryQuery) ([]domain.Payout, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Payout, 0, len(r.payouts))
	for _, payout := range r.payouts {
		if query.UserID != "" && payout.UserID != query.UserID {
			continue
		}
		items = append(items, payout)
	}
	slices.SortFunc(items, func(a, b domain.Payout) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	total := len(items)
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	if query.Offset >= len(items) {
		return []domain.Payout{}, total, nil
	}
	end := query.Offset + query.Limit
	if end > len(items) {
		end = len(items)
	}
	out := make([]domain.Payout, end-query.Offset)
	copy(out, items[query.Offset:end])
	return out, total, nil
}

type IdempotencyRepository struct {
	mu      sync.Mutex
	records map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[key]
	if !ok {
		return nil, nil
	}
	if now.After(record.ExpiresAt) {
		delete(r.records, key)
		return nil, nil
	}
	clone := record
	return &clone, nil
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.records[key]; ok && time.Now().UTC().Before(existing.ExpiresAt) {
		if existing.RequestHash != requestHash {
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
	record, ok := r.records[key]
	if !ok {
		return domain.ErrNotFound
	}
	record.ResponseCode = responseCode
	record.ResponseBody = slices.Clone(responseBody)
	if at.After(record.ExpiresAt) {
		record.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.records[key] = record
	return nil
}

type dedupRecord struct {
	EventType string
	ExpiresAt time.Time
}

type EventDedupRepository struct {
	mu      sync.Mutex
	records map[string]dedupRecord
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[eventID]
	if !ok {
		return false, nil
	}
	if now.After(record.ExpiresAt) {
		delete(r.records, eventID)
		return false, nil
	}
	return true, nil
}

func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[eventID] = dedupRecord{EventType: eventType, ExpiresAt: expiresAt}
	return nil
}

type OutboxRepository struct {
	mu      sync.Mutex
	records map[string]ports.OutboxRecord
	order   []string
}

func (r *OutboxRepository) Enqueue(_ context.Context, record ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[record.RecordID] = record
	r.order = append(r.order, record.RecordID)
	return nil
}

func (r *OutboxRepository) ListPending(_ context.Context, limit int) ([]ports.OutboxRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]ports.OutboxRecord, 0, limit)
	for _, id := range r.order {
		record, ok := r.records[id]
		if !ok || record.SentAt != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	record.SentAt = &at
	r.records[recordID] = record
	return nil
}
