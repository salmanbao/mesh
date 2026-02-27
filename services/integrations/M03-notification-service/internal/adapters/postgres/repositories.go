package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/ports"
)

type Repositories struct {
	Notifications *NotificationRepo
	Preferences   *PreferencesRepo
	Scheduled     *ScheduledRepo
	Idempotency   *IdempotencyRepo
	EventDedup    *EventDedupRepo
}

func NewRepositories() *Repositories {
	return &Repositories{
		Notifications: &NotificationRepo{rows: map[string]domain.Notification{}},
		Preferences:   &PreferencesRepo{rows: map[string]domain.Preferences{}},
		Scheduled:     &ScheduledRepo{rows: map[string]struct{}{}},
		Idempotency:   &IdempotencyRepo{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:    &EventDedupRepo{rows: map[string]time.Time{}},
	}
}

type NotificationRepo struct {
	mu   sync.Mutex
	rows map[string]domain.Notification
}

func (r *NotificationRepo) Create(_ context.Context, row domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rows[row.NotificationID]; exists {
		return domain.ErrConflict
	}
	r.rows[row.NotificationID] = row
	return nil
}
func (r *NotificationRepo) GetByID(_ context.Context, notificationID string) (domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[notificationID]
	if !ok {
		return domain.Notification{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *NotificationRepo) Update(_ context.Context, row domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.NotificationID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[row.NotificationID] = row
	return nil
}
func (r *NotificationRepo) ListByUserID(_ context.Context, userID string, filter domain.NotificationFilter) ([]domain.Notification, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.Notification, 0)
	for _, row := range r.rows {
		if row.UserID != userID {
			continue
		}
		if filter.Type != "" && row.Type != filter.Type {
			continue
		}
		switch filter.Status {
		case "unread":
			if !row.IsUnread() {
				continue
			}
		case "read":
			if !row.IsRead() || row.IsArchived() {
				continue
			}
		case "archived":
			if !row.IsArchived() {
				continue
			}
		case "", "all":
		default: // ignore unknown status at repo layer
		}
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := len(items)
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []domain.Notification{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return append([]domain.Notification(nil), items[start:end]...), total, nil
}
func (r *NotificationRepo) CountUnread(_ context.Context, userID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, row := range r.rows {
		if row.UserID == userID && row.IsUnread() {
			count++
		}
	}
	return count, nil
}

type PreferencesRepo struct {
	mu   sync.Mutex
	rows map[string]domain.Preferences
}

func (r *PreferencesRepo) GetByUserID(_ context.Context, userID string) (domain.Preferences, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[userID]
	if !ok {
		return domain.Preferences{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *PreferencesRepo) Upsert(_ context.Context, row domain.Preferences) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[row.UserID] = row
	return nil
}

type ScheduledRepo struct {
	mu   sync.Mutex
	rows map[string]struct{}
}

func (r *ScheduledRepo) Delete(_ context.Context, scheduledID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.rows[scheduledID]
	if ok {
		delete(r.rows, scheduledID)
	}
	return ok, nil
}

type IdempotencyRepo struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepo) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
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
	cpy := row
	return &cpy, nil
}
func (r *IdempotencyRepo) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
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
func (r *IdempotencyRepo) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
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

type EventDedupRepo struct {
	mu   sync.Mutex
	rows map[string]time.Time
}

func (r *EventDedupRepo) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
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
func (r *EventDedupRepo) MarkProcessed(_ context.Context, eventID, _ string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[eventID] = expiresAt
	return nil
}
