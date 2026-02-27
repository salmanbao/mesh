package postgres

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/ports"
)

type Repositories struct {
	Layouts       *LayoutRepository
	Views         *CustomViewRepository
	Preferences   *PreferenceRepository
	Invalidations *InvalidationRepository
	Cache         *DashboardCacheRepository
	Idempotency   *IdempotencyRepository
	EventDedup    *EventDedupRepository
	Outbox        *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Layouts:       &LayoutRepository{records: map[string]domain.DashboardLayout{}},
		Views:         &CustomViewRepository{records: map[string]domain.CustomView{}, byUser: map[string][]string{}},
		Preferences:   &PreferenceRepository{records: map[string]domain.UserPreference{}},
		Invalidations: &InvalidationRepository{records: make([]domain.CacheInvalidation, 0, 128)},
		Cache:         &DashboardCacheRepository{records: map[string]ports.CachedDashboard{}},
		Idempotency:   &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:    &EventDedupRepository{records: map[string]dedupRecord{}},
		Outbox:        &OutboxRepository{records: map[string]ports.OutboxRecord{}, order: make([]string, 0, 128)},
	}
}

type LayoutRepository struct {
	mu      sync.RWMutex
	records map[string]domain.DashboardLayout
}

func (r *LayoutRepository) key(userID, deviceType string) string {
	return strings.TrimSpace(userID) + "::" + strings.ToLower(strings.TrimSpace(deviceType))
}

func (r *LayoutRepository) GetCurrent(_ context.Context, userID, deviceType string) (domain.DashboardLayout, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	layout, ok := r.records[r.key(userID, deviceType)]
	if !ok {
		return domain.DashboardLayout{}, domain.ErrNotFound
	}
	return layout, nil
}

func (r *LayoutRepository) Save(_ context.Context, layout domain.DashboardLayout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[r.key(layout.UserID, layout.DeviceType)] = layout
	return nil
}

type CustomViewRepository struct {
	mu      sync.RWMutex
	records map[string]domain.CustomView
	byUser  map[string][]string
}

func (r *CustomViewRepository) Create(_ context.Context, view domain.CustomView) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[view.ViewID] = view
	r.byUser[view.UserID] = append(r.byUser[view.UserID], view.ViewID)
	return nil
}

func (r *CustomViewRepository) GetByID(_ context.Context, userID, viewID string) (domain.CustomView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	view, ok := r.records[viewID]
	if !ok || view.UserID != userID {
		return domain.CustomView{}, domain.ErrNotFound
	}
	return view, nil
}

func (r *CustomViewRepository) ListByUser(_ context.Context, userID string) ([]domain.CustomView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byUser[userID]
	out := make([]domain.CustomView, 0, len(ids))
	for _, id := range ids {
		if v, ok := r.records[id]; ok {
			out = append(out, v)
		}
	}
	slices.SortFunc(out, func(a, b domain.CustomView) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return out, nil
}

type PreferenceRepository struct {
	mu      sync.RWMutex
	records map[string]domain.UserPreference
}

func (r *PreferenceRepository) GetByUser(_ context.Context, userID string) (domain.UserPreference, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pref, ok := r.records[userID]
	if !ok {
		return domain.UserPreference{}, domain.ErrNotFound
	}
	return pref, nil
}

func (r *PreferenceRepository) Upsert(_ context.Context, preference domain.UserPreference) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[preference.UserID] = preference
	return nil
}

type InvalidationRepository struct {
	mu      sync.RWMutex
	records []domain.CacheInvalidation
}

func (r *InvalidationRepository) Add(_ context.Context, row domain.CacheInvalidation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, row)
	return nil
}

func (r *InvalidationRepository) ListByUser(_ context.Context, userID string, limit int) ([]domain.CacheInvalidation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := make([]domain.CacheInvalidation, 0, limit)
	for i := len(r.records) - 1; i >= 0; i-- {
		row := r.records[i]
		if row.UserID != userID {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

type DashboardCacheRepository struct {
	mu      sync.Mutex
	records map[string]ports.CachedDashboard
}

func (r *DashboardCacheRepository) Get(_ context.Context, cacheKey string, now time.Time) (*ports.CachedDashboard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[cacheKey]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.records, cacheKey)
		return nil, nil
	}
	clone := rec
	return &clone, nil
}

func (r *DashboardCacheRepository) Upsert(_ context.Context, item ports.CachedDashboard) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[item.CacheKey] = item
	return nil
}

func (r *DashboardCacheRepository) InvalidateByUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	needle := "dashboard:" + strings.TrimSpace(userID) + ":"
	for key := range r.records {
		if strings.HasPrefix(key, needle) {
			delete(r.records, key)
		}
	}
	return nil
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
	r.records[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return domain.ErrNotFound
	}
	rec.ResponseCode = responseCode
	rec.ResponseBody = slices.Clone(responseBody)
	if at.After(rec.ExpiresAt) {
		rec.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.records[key] = rec
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
	rec, ok := r.records[eventID]
	if !ok {
		return false, nil
	}
	if now.After(rec.ExpiresAt) {
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
		rec, ok := r.records[id]
		if !ok || rec.SentAt != nil {
			continue
		}
		out = append(out, rec)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	rec.SentAt = &at
	r.records[recordID] = rec
	return nil
}
