package postgres

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/ports"
)

type Repositories struct {
	Rewards     *RewardRepository
	Rollovers   *RolloverRepository
	Snapshots   *SnapshotRepository
	Audit       *AuditLogRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Rewards: &RewardRepository{
			rewards: make(map[string]domain.Reward),
		},
		Rollovers: &RolloverRepository{
			records: make(map[string]domain.RolloverBalance),
		},
		Snapshots: &SnapshotRepository{
			records: make(map[string]ports.SubmissionViewSnapshot),
		},
		Audit: &AuditLogRepository{
			records: make([]ports.AuditRecord, 0, 128),
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

type RewardRepository struct {
	mu      sync.RWMutex
	rewards map[string]domain.Reward
	order   []string
}

func (r *RewardRepository) Save(_ context.Context, reward domain.Reward) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rewards[reward.SubmissionID]; !exists {
		r.order = append(r.order, reward.SubmissionID)
	}
	r.rewards[reward.SubmissionID] = reward
	return nil
}

func (r *RewardRepository) GetBySubmissionID(_ context.Context, submissionID string) (domain.Reward, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reward, ok := r.rewards[submissionID]
	if !ok {
		return domain.Reward{}, domain.ErrNotFound
	}
	return reward, nil
}

func (r *RewardRepository) ListByUser(_ context.Context, userID string, limit, offset int) ([]domain.Reward, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Reward, 0, len(r.rewards))
	for _, reward := range r.rewards {
		if userID != "" && reward.UserID != userID {
			continue
		}
		items = append(items, reward)
	}
	slices.SortFunc(items, func(a, b domain.Reward) int {
		return b.CalculatedAt.Compare(a.CalculatedAt)
	})
	total := len(items)
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []domain.Reward{}, total, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	out := make([]domain.Reward, end-offset)
	copy(out, items[offset:end])
	return out, total, nil
}

type RolloverRepository struct {
	mu      sync.RWMutex
	records map[string]domain.RolloverBalance
}

func (r *RolloverRepository) GetByUser(_ context.Context, userID string) (domain.RolloverBalance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[userID]
	if !ok {
		return domain.RolloverBalance{UserID: userID, Balance: 0, UpdatedAt: time.Now().UTC()}, nil
	}
	return record, nil
}

func (r *RolloverRepository) Upsert(_ context.Context, balance domain.RolloverBalance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[balance.UserID] = balance
	return nil
}

type SnapshotRepository struct {
	mu      sync.RWMutex
	records map[string]ports.SubmissionViewSnapshot
}

func (r *SnapshotRepository) Upsert(_ context.Context, snapshot ports.SubmissionViewSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[snapshot.SubmissionID] = snapshot
	return nil
}

func (r *SnapshotRepository) Get(_ context.Context, submissionID string) (ports.SubmissionViewSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[submissionID]
	if !ok {
		return ports.SubmissionViewSnapshot{}, domain.ErrNotFound
	}
	return record, nil
}

type AuditLogRepository struct {
	mu      sync.Mutex
	records []ports.AuditRecord
}

func (r *AuditLogRepository) Append(_ context.Context, record ports.AuditRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
	return nil
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
