package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/ports"
)

type Config struct {
	ServiceName          string
	Version              string
	DefaultTTL           time.Duration
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

type Service struct {
	cfg Config

	cache       ports.CacheRepository
	metrics     ports.CacheMetricsRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	startedAt time.Time
	nowFn     func() time.Time
}

type Dependencies struct {
	Config Config

	Cache       ports.CacheRepository
	Metrics     ports.CacheMetricsRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M18-Cache-State-Management"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 15 * time.Minute
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
	now := time.Now().UTC()
	return &Service{
		cfg:         cfg,
		cache:       deps.Cache,
		metrics:     deps.Metrics,
		idempotency: deps.Idempotency,
		eventDedup:  deps.EventDedup,
		outbox:      deps.Outbox,
		startedAt:   now,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
