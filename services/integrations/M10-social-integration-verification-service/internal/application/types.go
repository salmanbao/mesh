package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ConnectInput struct {
	Provider string
	UserID   string
}

type CallbackInput struct {
	Provider string
	UserID   string
	Code     string
	State    string
	Handle   string
}

type RecordFollowersSyncInput struct {
	SocialAccountID string
	UserID          string
	FollowerCount   int
}

type PostValidationInput struct {
	UserID   string
	Platform string
	PostID   string
}

type ComplianceViolationInput struct {
	UserID   string
	Platform string
	PostID   string
	Reason   string
}

type ConnectResult struct {
	AuthURL string
	State   string
}

type Service struct {
	cfg Config

	accounts    ports.SocialAccountRepository
	metrics     ports.SocialMetricRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config Config

	Accounts    ports.SocialAccountRepository
	Metrics     ports.SocialMetricRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M10-Social-Integration-Verification-Service"
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
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	return &Service{
		cfg:         cfg,
		accounts:    deps.Accounts,
		metrics:     deps.Metrics,
		idempotency: deps.Idempotency,
		eventDedup:  deps.EventDedup,
		outbox:      deps.Outbox,
		domainEvents: deps.DomainEvents,
		analytics:   deps.Analytics,
		dlq:         deps.DLQ,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
