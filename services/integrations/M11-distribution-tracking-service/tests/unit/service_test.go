package unit

import (
	"context"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/adapters/events"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/application"
)

func TestRegisterAndPollEmitsTrackingMetricsUpdated(t *testing.T) {
	repos := postgres.NewRepositories()
	domainPub := eventadapter.NewMemoryDomainPublisher()
	svc := application.NewService(application.Dependencies{Posts: repos.Posts, Snapshots: repos.Snapshots, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup, Outbox: repos.Outbox, DomainEvents: domainPub, Config: application.Config{PollCadence: time.Nanosecond}})
	actor := application.Actor{SubjectID: "u1", Role: "user", IdempotencyKey: "idem-1"}
	post, _, err := svc.RegisterPost(context.Background(), actor, application.RegisterPostInput{UserID: "u1", Platform: "tiktok", PostURL: "https://tiktok.com/@user/video/123"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.RunPollCycle(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if err := svc.FlushOutbox(context.Background()); err != nil {
		t.Fatalf("flush outbox: %v", err)
	}
	events := domainPub.Events()
	if len(events) == 0 {
		t.Fatalf("expected emitted event")
	}
	if events[0].EventType != "tracking.metrics.updated" {
		t.Fatalf("unexpected event type: %s", events[0].EventType)
	}
	metricsPost, snaps, err := svc.GetTrackedPostMetrics(context.Background(), application.Actor{SubjectID: "u1", Role: "user"}, post.TrackedPostID)
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	if metricsPost.TrackedPostID == "" || len(snaps) == 0 {
		t.Fatalf("expected tracked post and snapshots")
	}
}
