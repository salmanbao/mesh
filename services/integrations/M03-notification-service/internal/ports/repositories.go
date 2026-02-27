package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/domain"
)

type NotificationRepository interface {
	Create(ctx context.Context, row domain.Notification) error
	GetByID(ctx context.Context, notificationID string) (domain.Notification, error)
	Update(ctx context.Context, row domain.Notification) error
	ListByUserID(ctx context.Context, userID string, filter domain.NotificationFilter) ([]domain.Notification, int, error)
	CountUnread(ctx context.Context, userID string) (int, error)
}

type PreferencesRepository interface {
	GetByUserID(ctx context.Context, userID string) (domain.Preferences, error)
	Upsert(ctx context.Context, row domain.Preferences) error
}

type ScheduledRepository interface {
	Delete(ctx context.Context, scheduledID string) (bool, error)
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}
