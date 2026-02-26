package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
)

type InvoiceQuery struct {
	Status        string
	DateFrom      *time.Time
	DateTo        *time.Time
	Limit         int
	Offset        int
	InvoiceNumber string
	CustomerEmail string
	MinAmount     float64
	MaxAmount     float64
}

type InvoiceRepository interface {
	NextInvoiceSequence(ctx context.Context, day time.Time) (int, error)
	Create(ctx context.Context, invoice domain.Invoice) error
	GetByID(ctx context.Context, invoiceID string) (domain.Invoice, error)
	Update(ctx context.Context, invoice domain.Invoice) error
	ListByCustomer(ctx context.Context, customerID string, query InvoiceQuery) ([]domain.Invoice, int, error)
	Search(ctx context.Context, query InvoiceQuery) ([]domain.Invoice, int, error)
	RecordEmailEvent(ctx context.Context, event domain.InvoiceEmailEvent) error
	RecordVoid(ctx context.Context, record domain.VoidHistory) error
	RecordPayment(ctx context.Context, payment domain.InvoicePayment) error
	CreatePayoutReceipt(ctx context.Context, receipt domain.PayoutReceipt) error
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
