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
	domainPublisher := eventadapter.NewMemoryDomainPublisher()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:                  "M41-Reward-Engine",
			EnableDomainEventConsumption: true,
			EnablePayoutEligibleEmission: true,
		},
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
		DomainEvents: domainPublisher,
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
		Config: application.Config{
			ServiceName:                  "M41-Reward-Engine",
			EnableDomainEventConsumption: true,
			EnablePayoutEligibleEmission: true,
		},
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

func TestHandleSubmissionViewLockedEventAllowsLegacyPartitionPath(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:                  "M41-Reward-Engine",
			EnableDomainEventConsumption: true,
			EnablePayoutEligibleEmission: true,
		},
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
		EventID:          "evt-legacy-submission-path",
		EventType:        domain.EventSubmissionViewLocked,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "submission_id",
		PartitionKey:     "sub-legacy-1",
		SourceService:    "M26-Submission-Service",
		TraceID:          "trace-legacy-submission-path",
		SchemaVersion:    "v1",
		Data: []byte(`{
			"submission_id":"sub-legacy-1",
			"user_id":"user-legacy-1",
			"campaign_id":"camp-legacy-1",
			"locked_views":1250,
			"locked_at":"2026-02-10T00:00:00Z"
		}`),
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("expected legacy partition path to be accepted, got: %v", err)
	}
}

func TestHandleTrackingMetricsEventAllowsLegacyPartitionPath(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:                  "M41-Reward-Engine",
			EnableDomainEventConsumption: true,
			EnablePayoutEligibleEmission: true,
		},
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
		EventID:          "evt-legacy-tracking-path",
		EventType:        domain.EventTrackingMetricsUpdated,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "tracked_post_id",
		PartitionKey:     "sub-legacy-track-1",
		SourceService:    "M11-Distribution-Tracking-Service",
		TraceID:          "trace-legacy-tracking-path",
		SchemaVersion:    "v1",
		Data: []byte(`{
			"tracked_post_id":"sub-legacy-track-1",
			"platform":"tiktok",
			"views":900,
			"likes":30,
			"shares":6,
			"comments":4,
			"polled_at":"2026-02-10T00:00:00Z"
		}`),
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("expected legacy tracking partition path to be accepted, got: %v", err)
	}
}

func TestPayoutEligibleEmissionCanBeDisabledByFlag(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	domainPublisher := eventadapter.NewMemoryDomainPublisher()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:                  "M41-Reward-Engine",
			EnableDomainEventConsumption: true,
			EnablePayoutEligibleEmission: false,
		},
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
		DomainEvents: domainPublisher,
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})

	_, err := svc.CalculateReward(context.Background(), application.Actor{
		SubjectID:      "user-flag",
		Role:           "user",
		IdempotencyKey: "reward:req:user-flag:sub-flag",
	}, application.CalculateRewardInput{
		UserID:                  "user-flag",
		SubmissionID:            "sub-flag",
		CampaignID:              "camp-flag",
		LockedViews:             2500,
		RatePer1K:               2.5,
		VerificationCompletedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("calculate reward: %v", err)
	}

	events := domainPublisher.Events()
	if len(events) == 0 {
		t.Fatalf("expected at least one domain event")
	}
	foundCalculated := false
	for _, evt := range events {
		if evt.EventType == domain.EventRewardCalculated {
			foundCalculated = true
		}
		if evt.EventType == domain.EventRewardPayoutEligible {
			t.Fatalf("did not expect %s when flag disabled", domain.EventRewardPayoutEligible)
		}
	}
	if !foundCalculated {
		t.Fatalf("expected %s event", domain.EventRewardCalculated)
	}
}

func TestDomainEventConsumptionCanBeDisabledByFlag(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:                  "M41-Reward-Engine",
			EnableDomainEventConsumption: false,
			EnablePayoutEligibleEmission: true,
		},
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
		EventID:          "evt-consumer-disabled",
		EventType:        domain.EventSubmissionViewLocked,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.submission_id",
		PartitionKey:     "sub-consumer-disabled",
		SourceService:    "M26-Submission-Service",
		TraceID:          "trace-consumer-disabled",
		SchemaVersion:    "v1",
		Data: []byte(`{
			"submission_id":"sub-consumer-disabled",
			"user_id":"user-consumer-disabled",
			"campaign_id":"camp-consumer-disabled",
			"locked_views":1500,
			"locked_at":"2026-02-10T00:00:00Z"
		}`),
	}
	if err := svc.HandleDomainEvent(context.Background(), event); err != nil {
		t.Fatalf("handle event with consumer disabled: %v", err)
	}

	if _, err := repos.Rewards.GetBySubmissionID(context.Background(), "sub-consumer-disabled"); err == nil {
		t.Fatalf("expected no reward write when domain consumption disabled")
	}
}
