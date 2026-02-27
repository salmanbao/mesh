package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/ports"
)

type Config struct {
	ServiceName          string
	Version              string
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

type ConnectAccountInput struct {
	UserID    string
	Platform  string
	Handle    string
	OAuthCode string
}

type ValidatePostInput struct {
	UserID   string
	Platform string
	PostID   string
}

type Service struct {
	cfg Config

	accounts    ports.SocialAccountRepository
	validations ports.PostValidationRepository
	metrics     ports.SocialMetricRepository
	ownerAPI    ports.SocialVerificationOwnerAPI
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository
	startedAt   time.Time
	nowFn       func() time.Time
}

type Dependencies struct {
	Config Config

	Accounts    ports.SocialAccountRepository
	Validations ports.PostValidationRepository
	Metrics     ports.SocialMetricRepository
	OwnerAPI    ports.SocialVerificationOwnerAPI
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M30-Social-Integration-Service"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
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
		accounts:    deps.Accounts,
		validations: deps.Validations,
		metrics:     deps.Metrics,
		ownerAPI:    deps.OwnerAPI,
		idempotency: deps.Idempotency,
		eventDedup:  deps.EventDedup,
		outbox:      deps.Outbox,
		startedAt:   now,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
