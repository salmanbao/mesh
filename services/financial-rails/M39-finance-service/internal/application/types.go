package application

import (
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	DefaultCurrency      string
	MaximumAmount        float64
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateTransactionInput struct {
	UserID                string
	CampaignID            string
	ProductID             string
	Provider              domain.PaymentProvider
	ProviderTransactionID string
	Amount                float64
	Currency              string
	TrafficSource         string
	UserTier              string
}

type CreateRefundInput struct {
	TransactionID string
	UserID        string
	Amount        float64
	Reason        string
}

type HandleWebhookInput struct {
	WebhookID             string
	Provider              string
	EventType             string
	ProviderEventID       string
	ProviderTransactionID string
	TransactionID         string
	UserID                string
	Amount                float64
	Currency              string
	Reason                string
}

type ListTransactionsOutput struct {
	Items      []domain.Transaction
	Pagination contracts.Pagination
}

type Service struct {
	cfg          Config
	transactions ports.TransactionRepository
	refunds      ports.RefundRepository
	balances     ports.BalanceRepository
	webhooks     ports.WebhookRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	outbox       ports.OutboxRepository

	auth           ports.AuthReader
	campaign       ports.CampaignReader
	contentLibrary ports.ContentLibraryReader
	escrow         ports.EscrowReader
	feeEngine      ports.FeeEngineReader
	product        ports.ProductReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Transactions ports.TransactionRepository
	Refunds      ports.RefundRepository
	Balances     ports.BalanceRepository
	Webhooks     ports.WebhookRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository

	Auth           ports.AuthReader
	Campaign       ports.CampaignReader
	ContentLibrary ports.ContentLibraryReader
	Escrow         ports.EscrowReader
	FeeEngine      ports.FeeEngineReader
	Product        ports.ProductReader

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M39-Finance-Service"
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
	if cfg.MaximumAmount <= 0 {
		cfg.MaximumAmount = 25000
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}

	return &Service{
		cfg:            cfg,
		transactions:   deps.Transactions,
		refunds:        deps.Refunds,
		balances:       deps.Balances,
		webhooks:       deps.Webhooks,
		idempotency:    deps.Idempotency,
		eventDedup:     deps.EventDedup,
		outbox:         deps.Outbox,
		auth:           deps.Auth,
		campaign:       deps.Campaign,
		contentLibrary: deps.ContentLibrary,
		escrow:         deps.Escrow,
		feeEngine:      deps.FeeEngine,
		product:        deps.Product,
		domainEvents:   deps.DomainEvents,
		analytics:      deps.Analytics,
		dlq:            deps.DLQ,
		nowFn:          time.Now().UTC,
	}
}
