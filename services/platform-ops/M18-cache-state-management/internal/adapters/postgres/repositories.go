package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/ports"
)

type Repositories struct {
	Cache       *CacheRepository
	Metrics     *CacheMetricsRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	cache := &CacheRepository{rows: map[string]domain.CacheEntry{}}
	metrics := &CacheMetricsRepository{}
	return &Repositories{
		Cache:       cache,
		Metrics:     metrics,
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]eventDedupRecord{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type CacheRepository struct {
	mu   sync.Mutex
	rows map[string]domain.CacheEntry
}

func (r *CacheRepository) Put(_ context.Context, row domain.CacheEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[strings.TrimSpace(row.Key)] = row
	return nil
}

func (r *CacheRepository) Get(_ context.Context, key string, now time.Time) (domain.CacheItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key = strings.TrimSpace(key)
	row, ok := r.rows[key]
	if !ok {
		return domain.CacheItem{Key: key, Found: false}, nil
	}
	if !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		delete(r.rows, key)
		return domain.CacheItem{Key: key, Found: false}, nil
	}
	ttl := 0
	if !row.ExpiresAt.IsZero() {
		ttl = int(row.ExpiresAt.Sub(now).Seconds())
		if ttl < 0 {
			ttl = 0
		}
	}
	return domain.CacheItem{Key: key, Value: append([]byte(nil), row.Value...), Found: true, TTLSeconds: ttl}, nil
}

func (r *CacheRepository) Delete(_ context.Context, key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key = strings.TrimSpace(key)
	if _, ok := r.rows[key]; !ok {
		return false, nil
	}
	delete(r.rows, key)
	return true, nil
}

func (r *CacheRepository) Invalidate(_ context.Context, keys []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := r.rows[key]; ok {
			delete(r.rows, key)
			count++
		}
	}
	return count, nil
}

func (r *CacheRepository) MemoryUsedBytes(context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var total int64
	for k, row := range r.rows {
		total += int64(len(k) + len(row.Value))
	}
	return total, nil
}

type CacheMetricsRepository struct {
	mu     sync.Mutex
	hits   int64
	misses int64
	evicts int64
	memory int64
}

func (r *CacheMetricsRepository) RecordHit(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hits++
	return nil
}
func (r *CacheMetricsRepository) RecordMiss(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.misses++
	return nil
}
func (r *CacheMetricsRepository) RecordEviction(_ context.Context, count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evicts += int64(count)
	return nil
}
func (r *CacheMetricsRepository) SetMemoryUsed(_ context.Context, bytes int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memory = bytes
	return nil
}
func (r *CacheMetricsRepository) Snapshot(context.Context) (domain.CacheMetrics, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return domain.CacheMetrics{Hits: r.hits, Misses: r.misses, Evictions: r.evicts, MemoryUsedBytes: r.memory}, nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(key)]
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
	if row, ok := r.rows[key]; ok && (row.ExpiresAt.IsZero() || time.Now().UTC().Before(row.ExpiresAt)) {
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
	row, ok := r.rows[key]
	if !ok {
		return domain.ErrNotFound
	}
	row.ResponseCode = responseCode
	row.ResponseBody = append([]byte(nil), responseBody...)
	if row.ExpiresAt.IsZero() {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.rows[key] = row
	return nil
}

type eventDedupRecord struct {
	EventType string
	ExpiresAt time.Time
}

type EventDedupRepository struct {
	mu   sync.Mutex
	rows map[string]eventDedupRecord
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(eventID)]
	if !ok {
		return false, nil
	}
	if !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		delete(r.rows, eventID)
		return false, nil
	}
	return true, nil
}

func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[strings.TrimSpace(eventID)] = eventDedupRecord{EventType: strings.TrimSpace(eventType), ExpiresAt: expiresAt}
	return nil
}

type OutboxRepository struct {
	mu    sync.Mutex
	rows  map[string]ports.OutboxRecord
	order []string
}

func (r *OutboxRepository) Enqueue(_ context.Context, row ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.RecordID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.RecordID] = row
	r.order = append(r.order, row.RecordID)
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
		row := r.rows[id]
		if row.SentAt != nil {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	row.SentAt = &at
	r.rows[recordID] = row
	return nil
}
