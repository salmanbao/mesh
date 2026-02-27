package unit

import (
	"context"
	"testing"

	eventadapter "github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/domain"
)

func newTestService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		RiskProfiles: repos.RiskProfiles,
		Escrow:       repos.Escrow,
		Disputes:     repos.Disputes,
		Evidence:     repos.Evidence,
		FraudFlags:   repos.FraudFlags,
		ReserveLogs:  repos.ReserveLogs,
		DebtLogs:     repos.DebtLogs,
		Suspensions:  repos.Suspensions,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		Auth:         grpcadapter.NewAuthClient("stub"),
		Profile:      grpcadapter.NewProfileClient("stub"),
		Fraud:        grpcadapter.NewFraudClient("stub"),
		Moderation:   grpcadapter.NewModerationClient("stub"),
		Resolution:   grpcadapter.NewResolutionClient("stub"),
		Reputation:   grpcadapter.NewReputationClient("stub"),
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
		Config:       application.Config{WebhookBearerToken: "test-secret"},
	})
	return svc, repos
}

func TestGetSellerRiskDashboard(t *testing.T) {
	svc, _ := newTestService()
	got, err := svc.GetSellerRiskDashboard(context.Background(), application.Actor{SubjectID: "seller_123", Role: "seller"})
	if err != nil {
		t.Fatalf("GetSellerRiskDashboard error: %v", err)
	}
	if got.RiskLevel == "" {
		t.Fatalf("expected risk level")
	}
	if got.ReserveStatus.PercentageHeld == 0 {
		t.Fatalf("expected reserve percentage to be populated")
	}
}

func TestFileDisputeIdempotency(t *testing.T) {
	svc, _ := newTestService()
	actor := application.Actor{SubjectID: "buyer_1", Role: "seller", IdempotencyKey: "idem-1"}
	input := application.FileDisputeInput{TransactionID: "txn_123", DisputeType: "refund_request", Reason: "not as described", BuyerClaim: "item differs"}

	first, err := svc.FileDispute(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first FileDispute error: %v", err)
	}
	second, err := svc.FileDispute(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second FileDispute error: %v", err)
	}
	if first.DisputeID != second.DisputeID {
		t.Fatalf("expected same dispute from idempotent replay, got %s vs %s", first.DisputeID, second.DisputeID)
	}

	_, err = svc.FileDispute(context.Background(), actor, application.FileDisputeInput{TransactionID: "txn_456", DisputeType: "refund_request", Reason: "other", BuyerClaim: "changed"})
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestHandleChargebackWebhookDedup(t *testing.T) {
	svc, repos := newTestService()
	input := application.ChargebackInput{
		EventID:          "evt_1",
		EventType:        "charge.dispute.created",
		OccurredAt:       "2026-02-01T00:00:00Z",
		SourceService:    "payment_provider",
		TraceID:          "trace_1",
		SchemaVersion:    "v1",
		PartitionKeyPath: "data.seller_id",
		PartitionKey:     "seller_abc",
		Amount:           20,
		ChargeID:         "ch_1",
		Currency:         "USD",
		DisputeReason:    "fraudulent",
		SellerID:         "seller_abc",
	}

	first, err := svc.HandleChargebackWebhook(context.Background(), "test-secret", input)
	if err != nil {
		t.Fatalf("first webhook error: %v", err)
	}
	if dup, _ := first["duplicate"].(bool); dup {
		t.Fatalf("first webhook should not be duplicate")
	}

	second, err := svc.HandleChargebackWebhook(context.Background(), "test-secret", input)
	if err != nil {
		t.Fatalf("second webhook error: %v", err)
	}
	if dup, _ := second["duplicate"].(bool); !dup {
		t.Fatalf("second webhook should be duplicate")
	}

	debts, err := repos.DebtLogs.ListBySeller(context.Background(), "seller_abc", 10)
	if err != nil {
		t.Fatalf("ListBySeller debt error: %v", err)
	}
	if len(debts) != 1 {
		t.Fatalf("expected 1 debt log, got %d", len(debts))
	}
}
