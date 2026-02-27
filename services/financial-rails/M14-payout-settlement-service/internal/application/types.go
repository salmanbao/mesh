package application

import (
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	DefaultCurrency      string
	InstantPayoutLimit   float64
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type RequestPayoutInput struct {
	UserID       string
	SubmissionID string
	Amount       float64
	Currency     string
	Method       domain.PayoutMethod
	ScheduledAt  time.Time
}

type HistoryOutput struct {
	Items      []domain.Payout
	Pagination contracts.Pagination
}

type Service struct {
	cfg         Config
	payouts     ports.PayoutRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	auth    ports.AuthReader
	profile ports.ProfileReader
	billing ports.BillingReader
	escrow  ports.EscrowReader
	risk    ports.RiskReader
	finance ports.FinanceReader
	reward  ports.RewardReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Payouts      ports.PayoutRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository
	Auth         ports.AuthReader
	Profile      ports.ProfileReader
	Billing      ports.BillingReader
	Escrow       ports.EscrowReader
	Risk         ports.RiskReader
	Finance      ports.FinanceReader
	Reward       ports.RewardReader
	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M14-Payout-Settlement-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.DefaultCurrency == "" {
		cfg.DefaultCurrency = "USD"
	}
	if cfg.InstantPayoutLimit <= 0 {
		cfg.InstantPayoutLimit = 10000
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	return &Service{
		cfg:          cfg,
		payouts:      deps.Payouts,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		auth:         deps.Auth,
		profile:      deps.Profile,
		billing:      deps.Billing,
		escrow:       deps.Escrow,
		risk:         deps.Risk,
		finance:      deps.Finance,
		reward:       deps.Reward,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		nowFn:        time.Now().UTC,
	}
}
