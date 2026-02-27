package unit

import (
	"context"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/domain"
)

func TestCalculateRewardIdempotency(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Rewards:      repos.Rewards,
		Rollovers:    repos.Rollovers,
		Snapshots:    repos.Snapshots,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		Auth:         grpcadapter.NewAuthClient(""),
		Campaign:     grpcadapter.NewCampaignClient(""),
		Voting:       grpcadapter.NewVotingClient(""),
		Tracking:     grpcadapter.NewTrackingClient(""),
		Submission:   grpcadapter.NewSubmissionClient(""),
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})

	actor := application.Actor{
		SubjectID:      "user-1",
		Role:           "user",
		IdempotencyKey: "reward:req:user-1:sub-1",
	}
	input := application.CalculateRewardInput{
		UserID:                  "user-1",
		SubmissionID:            "sub-1",
		CampaignID:              "camp-1",
		LockedViews:             1500,
		RatePer1K:               2.5,
		VerificationCompletedAt: time.Now().UTC(),
	}

	first, err := svc.CalculateReward(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first calculate reward: %v", err)
	}
	second, err := svc.CalculateReward(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second calculate reward: %v", err)
	}
	if first.SubmissionID != second.SubmissionID || first.CalculatedAt != second.CalculatedAt {
		t.Fatalf("expected same reward for idempotent replay")
	}
}

func TestHandleSubmissionViewLockedEventDedup(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Rewards:      repos.Rewards,
		Rollovers:    repos.Rollovers,
		Snapshots:    repos.Snapshots,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		Auth:         grpcadapter.NewAuthClient(""),
		Campaign:     grpcadapter.NewCampaignClient(""),
		Voting:       grpcadapter.NewVotingClient(""),
		Tracking:     grpcadapter.NewTrackingClient(""),
		Submission:   grpcadapter.NewSubmissionClient(""),
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})

	event := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        domain.EventSubmissionViewLocked,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.submission_id",
		PartitionKey:     "sub-1",
		SourceService:    "M26-Submission-Service",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		Data: []byte(`{
			"submission_id":"sub-1",
			"user_id":"user-1",
			"campaign_id":"camp-1",
			"locked_views":15750,
			"locked_at":"2026-02-10T00:00:00Z"
		}`),
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle first event: %v", err)
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle duplicate event: %v", err)
	}
}
