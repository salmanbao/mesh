package application

import (
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
)

type Config struct {
	ServiceName     string
	IdempotencyTTL  time.Duration
	EventDedupTTL   time.Duration
	DefaultTaxRate  float64
	DefaultCurrency string
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateInvoiceInput struct {
	CustomerID     string
	CustomerName   string
	CustomerEmail  string
	BillingAddress domain.Address
	InvoiceType    string
	LineItems      []domain.InvoiceLineItem
	DueDate        time.Time
	Notes          string
}

type VoidInvoiceInput struct {
	InvoiceID string
	Reason    string
}

type SendInvoiceInput struct {
	InvoiceID      string
	RecipientEmail string
}

type RefundInput struct {
	InvoiceID  string
	LineItemID string
	Amount     float64
	Reason     string
}

type Service struct {
	cfg          Config
	invoices     ports.InvoiceRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	auth         ports.AuthReader
	catalog      ports.CatalogReader
	fees         ports.FeeReader
	finance      ports.FinanceWriter
	subscription ports.SubscriptionReader
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Invoices     ports.InvoiceRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Auth         ports.AuthReader
	Catalog      ports.CatalogReader
	Fees         ports.FeeReader
	Finance      ports.FinanceWriter
	Subscription ports.SubscriptionReader
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M05-Billing-Service"
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
	if cfg.DefaultTaxRate <= 0 {
		cfg.DefaultTaxRate = 0.0825
	}
	return &Service{
		cfg:          cfg,
		invoices:     deps.Invoices,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		auth:         deps.Auth,
		catalog:      deps.Catalog,
		fees:         deps.Fees,
		finance:      deps.Finance,
		subscription: deps.Subscription,
		nowFn:        time.Now().UTC,
	}
}

type ListInvoicesOutput struct {
	Invoices   []domain.Invoice
	Pagination contracts.Pagination
}
