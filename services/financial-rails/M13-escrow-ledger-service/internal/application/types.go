package application

import (
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/ports"
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

type CreateHoldInput struct {
	CampaignID string
	CreatorID  string
	Amount     float64
}

type ReleaseInput struct {
	EscrowID string
	Amount   float64
}

type RefundInput struct {
	EscrowID string
	Amount   *float64
}

type Service struct {
	cfg Config
	holds       ports.EscrowHoldRepository
	ledger      ports.LedgerEntryRepository
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
	Holds       ports.EscrowHoldRepository
	Ledger      ports.LedgerEntryRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository
	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" { cfg.ServiceName = "M13-Escrow-Ledger-Service" }
	if cfg.IdempotencyTTL <= 0 { cfg.IdempotencyTTL = 7 * 24 * time.Hour }
	if cfg.EventDedupTTL <= 0 { cfg.EventDedupTTL = 7 * 24 * time.Hour }
	if cfg.ConsumerPollInterval <= 0 { cfg.ConsumerPollInterval = 2 * time.Second }
	if cfg.OutboxFlushBatchSize <= 0 { cfg.OutboxFlushBatchSize = 100 }
	return &Service{cfg: cfg, holds: deps.Holds, ledger: deps.Ledger, idempotency: deps.Idempotency, eventDedup: deps.EventDedup, outbox: deps.Outbox, domainEvents: deps.DomainEvents, analytics: deps.Analytics, dlq: deps.DLQ, nowFn: func() time.Time { return time.Now().UTC() }}
}
