package unit

import (
	"context"
	"testing"

	eventadapter "github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
)

func TestCreateTransactionIdempotency(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Transactions:   repos.Transactions,
		Refunds:        repos.Refunds,
		Balances:       repos.Balances,
		Webhooks:       repos.Webhooks,
		Idempotency:    repos.Idempotency,
		EventDedup:     repos.EventDedup,
		Outbox:         repos.Outbox,
		Auth:           grpcadapter.NewAuthClient(""),
		Campaign:       grpcadapter.NewCampaignClient(""),
		ContentLibrary: grpcadapter.NewContentLibraryClient(""),
		Escrow:         grpcadapter.NewEscrowClient(""),
		FeeEngine:      grpcadapter.NewFeeEngineClient(""),
		Product:        grpcadapter.NewProductClient(""),
		DomainEvents:   eventadapter.NewMemoryDomainPublisher(),
		Analytics:      eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:            eventadapter.NewLoggingDLQPublisher(),
	})

	actor := application.Actor{
		SubjectID:      "user-1",
		Role:           "user",
		IdempotencyKey: "txn:req:user-1:campaign-1",
	}
	input := application.CreateTransactionInput{
		UserID:        "user-1",
		CampaignID:    "campaign-1",
		ProductID:     "product-1",
		Provider:      domain.ProviderStripe,
		Amount:        125.25,
		Currency:      "USD",
		TrafficSource: "creator_brought",
		UserTier:      "free",
	}

	first, err := svc.CreateTransaction(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first create transaction: %v", err)
	}
	second, err := svc.CreateTransaction(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second create transaction: %v", err)
	}
	if first.TransactionID != second.TransactionID {
		t.Fatalf("expected same transaction for idempotent replay")
	}
}

func TestWebhookDedup(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Transactions:   repos.Transactions,
		Refunds:        repos.Refunds,
		Balances:       repos.Balances,
		Webhooks:       repos.Webhooks,
		Idempotency:    repos.Idempotency,
		EventDedup:     repos.EventDedup,
		Outbox:         repos.Outbox,
		Auth:           grpcadapter.NewAuthClient(""),
		Campaign:       grpcadapter.NewCampaignClient(""),
		ContentLibrary: grpcadapter.NewContentLibraryClient(""),
		Escrow:         grpcadapter.NewEscrowClient(""),
		FeeEngine:      grpcadapter.NewFeeEngineClient(""),
		Product:        grpcadapter.NewProductClient(""),
		DomainEvents:   eventadapter.NewMemoryDomainPublisher(),
		Analytics:      eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:            eventadapter.NewLoggingDLQPublisher(),
	})

	_, err := svc.HandleProviderWebhook(context.Background(), application.HandleWebhookInput{
		WebhookID:             "wh-1",
		Provider:              "stripe",
		EventType:             "payment_intent.succeeded",
		ProviderEventID:       "evt_1",
		ProviderTransactionID: "pi_1",
		TransactionID:         "",
		UserID:                "user-1",
		Amount:                99.99,
		Currency:              "USD",
	})
	if err != nil {
		t.Fatalf("handle first webhook: %v", err)
	}
	_, err = svc.HandleProviderWebhook(context.Background(), application.HandleWebhookInput{
		WebhookID:             "wh-1",
		Provider:              "stripe",
		EventType:             "payment_intent.succeeded",
		ProviderEventID:       "evt_1",
		ProviderTransactionID: "pi_1",
		TransactionID:         "",
		UserID:                "user-1",
		Amount:                99.99,
		Currency:              "USD",
	})
	if err != nil {
		t.Fatalf("handle duplicate webhook: %v", err)
	}
}
