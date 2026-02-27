package postgres

import (
	"context"
	"hash/fnv"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/ports"
)

type Repositories struct {
	Recommendations *RecommendationsRepository
	Feedback        *FeedbackRepository
	Overrides       *OverridesRepository
	Models          *ModelsRepository
	ABTests         *ABTestRepository
	Idempotency     *IdempotencyRepository
	EventDedup      *EventDedupRepository
	Outbox          *OutboxRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	models := &ModelsRepository{records: map[string]domain.RecommendationModel{}}
	_ = models.Upsert(context.Background(), domain.RecommendationModel{
		ModelID:   "mdl-default",
		Version:   "v2.1.0",
		Status:    "active",
		IsDefault: true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	return &Repositories{
		Recommendations: &RecommendationsRepository{batches: map[string]domain.RecommendationBatch{}},
		Feedback:        &FeedbackRepository{records: map[string]domain.FeedbackRecord{}},
		Overrides:       &OverridesRepository{records: map[string]domain.RecommendationOverride{}},
		Models:          models,
		ABTests:         &ABTestRepository{assignments: map[string]domain.ABTestAssignment{}},
		Idempotency:     &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:      &EventDedupRepository{records: map[string]dedupRecord{}},
		Outbox:          &OutboxRepository{records: map[string]ports.OutboxRecord{}},
	}
}

type RecommendationsRepository struct {
	mu      sync.RWMutex
	batches map[string]domain.RecommendationBatch
}

func (r *RecommendationsRepository) SaveBatch(_ context.Context, batch domain.RecommendationBatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := batch.UserID + "::" + batch.Role
	clone := batch
	clone.Recommendations = slices.Clone(batch.Recommendations)
	r.batches[key] = clone
	return nil
}

func (r *RecommendationsRepository) GetLatestBatch(_ context.Context, userID, role string) (domain.RecommendationBatch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	batch, ok := r.batches[userID+"::"+role]
	if !ok {
		return domain.RecommendationBatch{}, domain.ErrNotFound
	}
	clone := batch
	clone.Recommendations = slices.Clone(batch.Recommendations)
	return clone, nil
}

type FeedbackRepository struct {
	mu      sync.RWMutex
	records map[string]domain.FeedbackRecord
	order   []string
}

func (r *FeedbackRepository) Create(_ context.Context, row domain.FeedbackRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.FeedbackID]; ok {
		return domain.ErrConflict
	}
	r.records[row.FeedbackID] = row
	r.order = append(r.order, row.FeedbackID)
	return nil
}

func (r *FeedbackRepository) ListByUser(_ context.Context, userID string, limit int) ([]domain.FeedbackRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := make([]domain.FeedbackRecord, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.records[r.order[i]]
		if row.UserID != userID {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

type OverridesRepository struct {
	mu      sync.RWMutex
	records map[string]domain.RecommendationOverride
}

func (r *OverridesRepository) Upsert(_ context.Context, row domain.RecommendationOverride) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.OverrideID] = row
	return nil
}

func (r *OverridesRepository) ListActive(_ context.Context, role string, now time.Time) ([]domain.RecommendationOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.RecommendationOverride, 0, len(r.records))
	for _, row := range r.records {
		if !row.Active {
			continue
		}
		if row.EndAt != nil && now.After(*row.EndAt) {
			continue
		}
		if row.Scope == "global" || row.Scope == "all" {
			out = append(out, row)
			continue
		}
		if row.Scope == "role_based" || row.Scope == "role" {
			if role == "" || stringsEqualFoldTrim(row.ScopeValue, role) {
				out = append(out, row)
			}
			continue
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *OverridesRepository) GetByID(_ context.Context, overrideID string) (domain.RecommendationOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[overrideID]
	if !ok {
		return domain.RecommendationOverride{}, domain.ErrNotFound
	}
	return row, nil
}

type ModelsRepository struct {
	mu      sync.RWMutex
	records map[string]domain.RecommendationModel
}

func (r *ModelsRepository) GetDefault(_ context.Context) (domain.RecommendationModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, row := range r.records {
		if row.IsDefault {
			return row, nil
		}
	}
	return domain.RecommendationModel{}, domain.ErrNotFound
}

func (r *ModelsRepository) Upsert(_ context.Context, row domain.RecommendationModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row.IsDefault {
		for id, model := range r.records {
			model.IsDefault = false
			r.records[id] = model
		}
	}
	r.records[row.ModelID] = row
	return nil
}

type ABTestRepository struct {
	mu          sync.Mutex
	assignments map[string]domain.ABTestAssignment
}

func (r *ABTestRepository) GetOrAssign(_ context.Context, userID string, now time.Time) (domain.ABTestAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.assignments[userID]; ok {
		return row, nil
	}
	variants := []string{domain.ABVariantControl, domain.ABVariantMLDriven, domain.ABVariantHybrid, domain.ABVariantDiversityFirst}
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID + "::rec-default"))
	variant := variants[int(h.Sum32())%len(variants)]
	row := domain.ABTestAssignment{
		AssignmentID: uuid.NewString(),
		TestID:       "rec-default",
		UserID:       userID,
		Variant:      variant,
		AssignedAt:   now,
	}
	r.assignments[userID] = row
	return row, nil
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

func stringsEqualFoldTrim(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
