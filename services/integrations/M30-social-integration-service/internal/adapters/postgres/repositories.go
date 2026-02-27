package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/ports"
)

type Repositories struct {
	Accounts    *SocialAccountRepository
	Validations *PostValidationRepository
	Metrics     *SocialMetricRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Accounts:    &SocialAccountRepository{byID: map[string]domain.SocialAccount{}, byUserProvider: map[string]string{}},
		Validations: &PostValidationRepository{byKey: map[string]domain.PostValidation{}},
		Metrics:     &SocialMetricRepository{rows: []domain.SocialMetric{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]eventDedupRow{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

func normalizeProvider(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "x" {
		return "twitter"
	}
	return v
}

func userProviderKey(userID, provider string) string {
	return strings.TrimSpace(userID) + ":" + normalizeProvider(provider)
}

func validationKey(userID, platform, postID string) string {
	return strings.TrimSpace(userID) + ":" + normalizeProvider(platform) + ":" + strings.TrimSpace(postID)
}

type SocialAccountRepository struct {
	mu             sync.Mutex
	byID           map[string]domain.SocialAccount
	byUserProvider map[string]string
}

func (r *SocialAccountRepository) Create(_ context.Context, row domain.SocialAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.SocialAccountID]; ok {
		return domain.ErrConflict
	}
	key := userProviderKey(row.UserID, row.Platform)
	if _, ok := r.byUserProvider[key]; ok {
		return domain.ErrConflict
	}
	r.byID[row.SocialAccountID] = row
	r.byUserProvider[key] = row.SocialAccountID
	return nil
}

func (r *SocialAccountRepository) GetByID(_ context.Context, socialAccountID string) (domain.SocialAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(socialAccountID)]
	if !ok {
		return domain.SocialAccount{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *SocialAccountRepository) Update(_ context.Context, row domain.SocialAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.byID[row.SocialAccountID]
	if !ok {
		return domain.ErrNotFound
	}
	delete(r.byUserProvider, userProviderKey(old.UserID, old.Platform))
	r.byID[row.SocialAccountID] = row
	r.byUserProvider[userProviderKey(row.UserID, row.Platform)] = row.SocialAccountID
	return nil
}

func (r *SocialAccountRepository) ListByUserID(_ context.Context, userID string) ([]domain.SocialAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	uid := strings.TrimSpace(userID)
	out := make([]domain.SocialAccount, 0)
	for _, row := range r.byID {
		if row.UserID == uid {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ConnectedAt.After(out[j].ConnectedAt)
	})
	return out, nil
}

func (r *SocialAccountRepository) GetByUserProvider(_ context.Context, userID, provider string) (domain.SocialAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byUserProvider[userProviderKey(userID, provider)]
	if !ok {
		return domain.SocialAccount{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok {
		return domain.SocialAccount{}, domain.ErrNotFound
	}
	return row, nil
}

type PostValidationRepository struct {
	mu    sync.Mutex
	byKey map[string]domain.PostValidation
}

func (r *PostValidationRepository) Create(ctx context.Context, row domain.PostValidation) error {
	if _, err := r.GetByUserPlatformPost(ctx, row.UserID, row.Platform, row.PostID); err == nil {
		return domain.ErrConflict
	}
	return r.UpsertByUserPlatformPost(ctx, row)
}

func (r *PostValidationRepository) UpsertByUserPlatformPost(_ context.Context, row domain.PostValidation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byKey[validationKey(row.UserID, row.Platform, row.PostID)] = row
	return nil
}

func (r *PostValidationRepository) GetByUserPlatformPost(_ context.Context, userID, platform, postID string) (domain.PostValidation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byKey[validationKey(userID, platform, postID)]
	if !ok {
		return domain.PostValidation{}, domain.ErrNotFound
	}
	return row, nil
}

type SocialMetricRepository struct {
	mu   sync.Mutex
	rows []domain.SocialMetric
}

func (r *SocialMetricRepository) Append(_ context.Context, row domain.SocialMetric) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}

func (r *SocialMetricRepository) LatestByUserPlatform(_ context.Context, userID, platform string) (domain.SocialMetric, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	uid := strings.TrimSpace(userID)
	plat := normalizeProvider(platform)
	for i := len(r.rows) - 1; i >= 0; i-- {
		if r.rows[i].UserID == uid && normalizeProvider(r.rows[i].Platform) == plat {
			return r.rows[i], nil
		}
	}
	return domain.SocialMetric{}, domain.ErrNotFound
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

type eventDedupRow struct {
	EventID   string
	EventType string
	ExpiresAt time.Time
}

type EventDedupRepository struct {
	mu   sync.Mutex
	rows map[string]eventDedupRow
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
	r.rows[strings.TrimSpace(eventID)] = eventDedupRow{EventID: strings.TrimSpace(eventID), EventType: strings.TrimSpace(eventType), ExpiresAt: expiresAt}
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
