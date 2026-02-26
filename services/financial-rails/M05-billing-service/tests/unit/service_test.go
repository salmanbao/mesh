package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
)

func TestCreateInvoiceIdempotency(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Invoices:     repos.Invoices,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Auth:         stubAuth{},
		Catalog:      stubCatalog{},
		Fees:         stubFees{},
		Finance:      stubFinance{},
		Subscription: stubSubscription{},
	})

	actor := application.Actor{
		SubjectID:      "admin-user",
		Role:           "admin",
		IdempotencyKey: "invoice-create:test",
	}
	input := application.CreateInvoiceInput{
		CustomerID:    "customer-1",
		CustomerName:  "Customer One",
		CustomerEmail: "customer1@example.com",
		BillingAddress: domain.Address{
			Line1:      "123 Main",
			City:       "SF",
			State:      "CA",
			PostalCode: "94105",
			Country:    "US",
		},
		InvoiceType: "invoice",
		LineItems: []domain.InvoiceLineItem{
			{Description: "Item", Quantity: 1, UnitPrice: 10},
		},
		DueDate: time.Now().UTC().Add(24 * time.Hour),
	}

	first, err := svc.CreateInvoice(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.CreateInvoice(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.InvoiceID != second.InvoiceID {
		t.Fatalf("expected same invoice id for idempotent replay")
	}
}

func TestHandleDomainEventDedup(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Invoices:     repos.Invoices,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Auth:         stubAuth{},
		Catalog:      stubCatalog{},
		Fees:         stubFees{},
		Finance:      stubFinance{},
		Subscription: stubSubscription{},
	})

	event := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "payout.paid",
		EventClass:       "domain",
		OccurredAt:       time.Now().UTC(),
		PartitionKey:     "p1",
		PartitionKeyPath: "data.payout_id",
		SourceService:    "m14-payout-settlement-service",
		TraceID:          "trace-1",
		SchemaVersion:    "1.0",
		Data:             []byte(`{"payout_id":"p1","creator_id":"c1","gross_amount":100,"fee_amount":10,"net_amount":90,"currency":"USD","paid_at":"2026-01-01T00:00:00Z"}`),
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle first event: %v", err)
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle duplicate event: %v", err)
	}
}

type stubAuth struct{}
type stubCatalog struct{}
type stubFees struct{}
type stubFinance struct{}
type stubSubscription struct{}

func (s stubAuth) GetUser(_ context.Context, userID string) (ports.UserIdentity, error) {
	return ports.UserIdentity{UserID: userID}, nil
}

func (s stubCatalog) GetSource(_ context.Context, sourceType, sourceID string) error {
	_ = sourceType
	_ = sourceID
	return nil
}

func (s stubFees) GetFeeRate(_ context.Context, sourceType string) (float64, error) {
	_ = sourceType
	return 0.08, nil
}

func (s stubFinance) RecordTransaction(_ context.Context, transactionType, invoiceID string, amount float64, currency string) error {
	_ = transactionType
	_ = invoiceID
	_ = amount
	_ = currency
	return nil
}

func (s stubSubscription) ValidateSubscription(_ context.Context, customerID string) error {
	_ = customerID
	return nil
}
