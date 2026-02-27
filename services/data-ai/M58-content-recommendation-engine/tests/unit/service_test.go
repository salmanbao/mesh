package unit

import (
	"context"
	"testing"
	"time"

	eventsadapter "github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/domain"
)

func newService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:          "M58-Content-Recommendation-Engine",
			IdempotencyTTL:       7 * 24 * time.Hour,
			EventDedupTTL:        7 * 24 * time.Hour,
			OutboxFlushBatchSize: 100,
			RecommendationTTL:    time.Hour,
		},
		Recommendations:   repos.Recommendations,
		Feedback:          repos.Feedback,
		Overrides:         repos.Overrides,
		Models:            repos.Models,
		ABTests:           repos.ABTests,
		Idempotency:       repos.Idempotency,
		EventDedup:        repos.EventDedup,
		Outbox:            repos.Outbox,
		CampaignDiscovery: grpcadapter.NewCampaignDiscoveryClient(""),
		DomainEvents:      eventsadapter.NewMemoryDomainPublisher(),
		Analytics:         eventsadapter.NewMemoryAnalyticsPublisher(),
		DLQ:               eventsadapter.NewLoggingDLQPublisher(),
	}), repos
}

func TestGetRecommendationsCachesAndQueuesOutbox(t *testing.T) {
	t.Parallel()
	svc, repos := newService()
	actor := application.Actor{SubjectID: "user-1", Role: "clipper", RequestID: "req-1"}

	first, err := svc.GetRecommendations(context.Background(), actor, application.GetRecommendationsInput{Limit: 10})
	if err != nil {
		t.Fatalf("first get recommendations: %v", err)
	}
	if first.Meta.CacheHit {
		t.Fatalf("expected first response to be cache miss")
	}
	pending, err := repos.Outbox.ListPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("list outbox pending: %v", err)
	}
	if len(pending) == 0 {
		t.Fatalf("expected recommendation.generated event in outbox")
	}

	second, err := svc.GetRecommendations(context.Background(), actor, application.GetRecommendationsInput{Limit: 10})
	if err != nil {
		t.Fatalf("second get recommendations: %v", err)
	}
	if !second.Meta.CacheHit {
		t.Fatalf("expected cached recommendations on second call")
	}
}

func TestRecordFeedbackIdempotency(t *testing.T) {
	t.Parallel()
	svc, _ := newService()
	actor := application.Actor{SubjectID: "user-1", Role: "clipper", RequestID: "req-2", IdempotencyKey: "idem-feedback-1"}
	input := application.FeedbackInput{
		EventID:          "evt-1",
		EventType:        domain.FeedbackEventSubmission,
		OccurredAt:       time.Now().UTC().Format(time.RFC3339),
		SourceService:    "web",
		TraceID:          "trace-1",
		SchemaVersion:    "1.0",
		PartitionKeyPath: "data.entity_id",
		PartitionKey:     "cmp_1",
		EntityID:         "cmp_1",
	}
	first, err := svc.RecordFeedback(context.Background(), actor, "rec-1", input)
	if err != nil {
		t.Fatalf("first record feedback: %v", err)
	}
	second, err := svc.RecordFeedback(context.Background(), actor, "rec-1", input)
	if err != nil {
		t.Fatalf("second record feedback: %v", err)
	}
	if first.FeedbackID != second.FeedbackID {
		t.Fatalf("expected idempotent replay to return same feedback id")
	}
}

func TestApplyOverrideRequiresAdmin(t *testing.T) {
	t.Parallel()
	svc, _ := newService()
	_, err := svc.ApplyOverride(context.Background(), application.Actor{SubjectID: "user-1", Role: "clipper", IdempotencyKey: "idem-ov-1"}, application.OverrideInput{
		OverrideType: domain.OverrideTypePromoteCampaign,
		EntityID:     "cmp_1",
		Scope:        "role_based",
		ScopeValue:   "clipper",
		Multiplier:   1.5,
		Reason:       "test",
	})
	if err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
