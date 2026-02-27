package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
	PollCadence          time.Duration
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ValidatePostInput struct{ UserID, Platform, PostURL string }
type RegisterPostInput struct{ UserID, Platform, PostURL, DistributionItemID, CampaignID string }

type Service struct {
	cfg          Config
	posts        ports.TrackedPostRepository
	snapshots    ports.MetricSnapshotRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	outbox       ports.OutboxRepository
	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Posts        ports.TrackedPostRepository
	Snapshots    ports.MetricSnapshotRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository
	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M11-Distribution-Tracking-Service"
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
	if cfg.PollCadence <= 0 {
		cfg.PollCadence = 6 * time.Hour
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	return &Service{cfg: cfg, posts: deps.Posts, snapshots: deps.Snapshots, idempotency: deps.Idempotency, eventDedup: deps.EventDedup, outbox: deps.Outbox, domainEvents: deps.DomainEvents, analytics: deps.Analytics, dlq: deps.DLQ, nowFn: func() time.Time { return time.Now().UTC() }}
}
