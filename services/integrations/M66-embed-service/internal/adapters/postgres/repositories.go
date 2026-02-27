package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/ports"
)

type Repositories struct {
	Settings     *EmbedSettingsRepository
	Cache        *EmbedCacheRepository
	Impressions  *ImpressionRepository
	Interactions *InteractionRepository
	Idempotency  *IdempotencyRepository
	EventDedup   *EventDedupRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Settings:     &EmbedSettingsRepository{rows: map[string]domain.EmbedSettings{}},
		Cache:        &EmbedCacheRepository{rows: map[string]domain.EmbedCache{}},
		Impressions:  &ImpressionRepository{rows: []domain.Impression{}},
		Interactions: &InteractionRepository{rows: []domain.Interaction{}},
		Idempotency:  &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:   &EventDedupRepository{rows: map[string]time.Time{}},
	}
}

func settingsKey(entityType, entityID string) string { return entityType + ":" + entityID }

type EmbedSettingsRepository struct {
	mu   sync.Mutex
	rows map[string]domain.EmbedSettings
}

func (r *EmbedSettingsRepository) GetByEntity(_ context.Context, entityType, entityID string) (domain.EmbedSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[settingsKey(entityType, entityID)]
	if !ok {
		return domain.EmbedSettings{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *EmbedSettingsRepository) Upsert(_ context.Context, row domain.EmbedSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[settingsKey(row.EntityType, row.EntityID)] = row
	return nil
}

type EmbedCacheRepository struct {
	mu   sync.Mutex
	rows map[string]domain.EmbedCache
}

func (r *EmbedCacheRepository) Get(_ context.Context, cacheKey string, now time.Time) (domain.EmbedCache, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[cacheKey]
	if !ok {
		return domain.EmbedCache{}, domain.ErrNotFound
	}
	if !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		delete(r.rows, cacheKey)
		return domain.EmbedCache{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *EmbedCacheRepository) Put(_ context.Context, row domain.EmbedCache) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[row.CacheKey] = row
	return nil
}
func (r *EmbedCacheRepository) DeleteByEntity(_ context.Context, entityType, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range r.rows {
		if v.EntityType == entityType && v.EntityID == entityID {
			delete(r.rows, k)
		}
	}
	return nil
}

type ImpressionRepository struct {
	mu   sync.Mutex
	rows []domain.Impression
}

func (r *ImpressionRepository) Append(_ context.Context, row domain.Impression) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *ImpressionRepository) CountByIPSince(_ context.Context, ipMasked string, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, row := range r.rows {
		if row.IPAnonymized == ipMasked && !row.OccurredAt.Before(since) {
			count++
		}
	}
	return count, nil
}
func (r *ImpressionRepository) CountByEntityReferrerSince(_ context.Context, entityType, entityID, referrerDomain string, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, row := range r.rows {
		if row.EntityType == entityType && row.EntityID == entityID && row.ReferrerDomain == referrerDomain && !row.OccurredAt.Before(since) {
			count++
		}
	}
	return count, nil
}
func (r *ImpressionRepository) ListByEntityRange(_ context.Context, entityType, entityID string, from, to *time.Time) ([]domain.Impression, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Impression{}
	for _, row := range r.rows {
		if row.EntityType != entityType || row.EntityID != entityID {
			continue
		}
		if from != nil && row.OccurredAt.Before(*from) {
			continue
		}
		if to != nil && row.OccurredAt.After(*to) {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	return out, nil
}

type InteractionRepository struct {
	mu   sync.Mutex
	rows []domain.Interaction
}

func (r *InteractionRepository) Append(_ context.Context, row domain.Interaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *InteractionRepository) ListByEntityRange(_ context.Context, entityType, entityID string, from, to *time.Time) ([]domain.Interaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Interaction{}
	for _, row := range r.rows {
		if row.EntityType != entityType || row.EntityID != entityID {
			continue
		}
		if from != nil && row.OccurredAt.Before(*from) {
			continue
		}
		if to != nil && row.OccurredAt.After(*to) {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
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
