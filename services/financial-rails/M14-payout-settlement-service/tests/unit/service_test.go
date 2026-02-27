package unit

import (
	"context"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
)

func TestRequestPayoutIdempotency(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Payouts:      repos.Payouts,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		Auth:         grpcadapter.NewAuthClient(""),
		Profile:      grpcadapter.NewProfileClient(""),
		Billing:      grpcadapter.NewBillingClient(""),
		Escrow:       grpcadapter.NewEscrowClient(""),
		Risk:         grpcadapter.NewRiskClient(""),
		Finance:      grpcadapter.NewFinanceClient(""),
		Reward:       grpcadapter.NewRewardClient(""),
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})

	actor := application.Actor{
		SubjectID:      "user-1",
		Role:           "user",
		IdempotencyKey: "pay:req:user-1:sub-1",
	}
	input := application.RequestPayoutInput{
		UserID:       "user-1",
		SubmissionID: "sub-1",
		Amount:       125.25,
		Currency:     "USD",
		Method:       domain.PayoutMethodStandard,
		ScheduledAt:  time.Now().UTC(),
	}

	first, err := svc.RequestPayout(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first request payout: %v", err)
	}
	second, err := svc.RequestPayout(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second request payout: %v", err)
	}
	if first.PayoutID != second.PayoutID {
		t.Fatalf("expected same payout for idempotent replay")
	}
}

func TestHandleRewardEligibleEventDedup(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Payouts:      repos.Payouts,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		Auth:         grpcadapter.NewAuthClient(""),
		Profile:      grpcadapter.NewProfileClient(""),
		Billing:      grpcadapter.NewBillingClient(""),
		Escrow:       grpcadapter.NewEscrowClient(""),
		Risk:         grpcadapter.NewRiskClient(""),
		Finance:      grpcadapter.NewFinanceClient(""),
		Reward:       grpcadapter.NewRewardClient(""),
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})

	event := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        domain.EventRewardPayoutEligible,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.submission_id",
		PartitionKey:     "sub-1",
		SourceService:    "M41-Reward-Engine",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		Data: []byte(`{
			"submission_id":"sub-1",
			"user_id":"user-1",
			"campaign_id":"camp-1",
			"locked_views":1000,
			"rate_per_1k":10,
			"gross_amount":10,
			"eligible_at":"2026-02-10T00:00:00Z"
		}`),
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle first event: %v", err)
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle duplicate event: %v", err)
	}
}
