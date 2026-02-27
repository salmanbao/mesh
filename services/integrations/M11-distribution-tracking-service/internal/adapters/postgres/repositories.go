package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/ports"
)

type Repositories struct {
	Posts       *TrackedPostRepository
	Snapshots   *MetricSnapshotRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Posts:       &TrackedPostRepository{rows: map[string]domain.TrackedPost{}},
		Snapshots:   &MetricSnapshotRepository{rows: map[string][]domain.MetricSnapshot{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}},
	}
}

type TrackedPostRepository struct {
	mu   sync.Mutex
	rows map[string]domain.TrackedPost
}

func (r *TrackedPostRepository) Create(_ context.Context, row domain.TrackedPost) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.TrackedPostID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.TrackedPostID] = row
	return nil
}
func (r *TrackedPostRepository) GetByID(_ context.Context, id string) (domain.TrackedPost, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if !ok {
		return domain.TrackedPost{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *TrackedPostRepository) Update(_ context.Context, row domain.TrackedPost) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.TrackedPostID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[row.TrackedPostID] = row
	return nil
}
func (r *TrackedPostRepository) FindByUserPlatformURL(_ context.Context, userID, platform, postURL string) (domain.TrackedPost, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.UserID == userID && row.Platform == platform && row.PostURL == postURL {
			return row, nil
		}
	}
	return domain.TrackedPost{}, domain.ErrNotFound
}
func (r *TrackedPostRepository) ListPollCandidates(_ context.Context, before time.Time, limit int) ([]domain.TrackedPost, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.TrackedPost, 0)
	for _, row := range r.rows {
		if row.Status == domain.TrackedPostStatusArchived {
			continue
		}
		if row.LastPolledAt == nil || row.LastPolledAt.Before(before) {
			items = append(items, row)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return append([]domain.TrackedPost(nil), items...), nil
}

type MetricSnapshotRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.MetricSnapshot
}

func (r *MetricSnapshotRepository) Append(_ context.Context, row domain.MetricSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[row.TrackedPostID] = append(r.rows[row.TrackedPostID], row)
	return nil
}
func (r *MetricSnapshotRepository) ListByTrackedPostID(_ context.Context, id string) ([]domain.MetricSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.MetricSnapshot(nil), r.rows[id]...), nil
}
func (r *MetricSnapshotRepository) LatestByTrackedPostID(_ context.Context, id string) (domain.MetricSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rows := r.rows[id]
	if len(rows) == 0 {
		return domain.MetricSnapshot{}, domain.ErrNotFound
	}
	return rows[len(rows)-1], nil
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

type OutboxRepository struct {
	mu   sync.Mutex
	rows map[string]ports.OutboxRecord
}

func (r *OutboxRepository) Enqueue(_ context.Context, record ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[record.RecordID] = record
	return nil
}
func (r *OutboxRepository) ListPending(_ context.Context, limit int) ([]ports.OutboxRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]ports.OutboxRecord, 0)
	for _, row := range r.rows {
		if row.SentAt == nil {
			items = append(items, row)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return append([]ports.OutboxRecord(nil), items...), nil
}
func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	t := at
	row.SentAt = &t
	r.rows[recordID] = row
	return nil
}
