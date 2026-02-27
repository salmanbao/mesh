package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ListNotificationsInput struct {
	UserID string
	Type   string
	Status string
	Limit  int
	Cursor string
}

type BulkActionInput struct {
	UserID          string
	Action          string
	NotificationIDs []string
}

type UpdatePreferencesInput struct {
	UserID            string
	EmailEnabled      *bool
	PushEnabled       *bool
	SMSEnabled        *bool
	InAppEnabled      *bool
	QuietHoursEnabled *bool
	QuietHoursStart   string
	QuietHoursEnd     string
	QuietHoursTZ      string
	Language          string
	BatchingEnabled   *bool
	MutedTypes        []string
}

type Service struct {
	cfg           Config
	notifications ports.NotificationRepository
	preferences   ports.PreferencesRepository
	scheduled     ports.ScheduledRepository
	idempotency   ports.IdempotencyRepository
	eventDedup    ports.EventDedupRepository
	nowFn         func() time.Time
}

type Dependencies struct {
	Config        Config
	Notifications ports.NotificationRepository
	Preferences   ports.PreferencesRepository
	Scheduled     ports.ScheduledRepository
	Idempotency   ports.IdempotencyRepository
	EventDedup    ports.EventDedupRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M03-Notification-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.ConsumerPollInterval <= 0 {
		cfg.ConsumerPollInterval = 2 * time.Second
	}
	return &Service{
		cfg:           cfg,
		notifications: deps.Notifications,
		preferences:   deps.Preferences,
		scheduled:     deps.Scheduled,
		idempotency:   deps.Idempotency,
		eventDedup:    deps.EventDedup,
		nowFn:         func() time.Time { return time.Now().UTC() },
	}
}
