package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/adapters/events"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Config:        application.Config{ServiceName: "M55-Dashboard-Service", IdempotencyTTL: 7 * 24 * time.Hour, EventDedupTTL: 7 * 24 * time.Hour, DashboardCacheTTL: 5 * time.Minute},
		Layouts:       repos.Layouts,
		Views:         repos.Views,
		Preferences:   repos.Preferences,
		Invalidations: repos.Invalidations,
		Cache:         repos.Cache,
		Idempotency:   repos.Idempotency,
		EventDedup:    repos.EventDedup,
		Outbox:        repos.Outbox,
		Profile:       grpc.NewProfileClient(""),
		Billing:       grpc.NewBillingClient(""),
		Content:       grpc.NewContentClient(""),
		Escrow:        grpc.NewEscrowClient(""),
		Onboarding:    grpc.NewOnboardingClient(""),
		Finance:       grpc.NewFinanceClient(""),
		Rewards:       grpc.NewRewardClient(""),
		Gamification:  grpc.NewGamificationClient(""),
		Analytics:     grpc.NewAnalyticsClient(""),
		Products:      grpc.NewProductClient(""),
		DLQ:           events.NewLoggingDLQPublisher(),
	})
}

func TestSaveLayoutIdempotency(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "user-1", Role: "creator", RequestID: "r1", IdempotencyKey: "layout-key-1"}
	input := application.SaveLayoutInput{
		DeviceType: "web",
		Items: []application.LayoutItemInput{
			{WidgetID: "earnings", Position: 0, Visible: true, Size: "2x3"},
			{WidgetID: "campaigns", Position: 1, Visible: true, Size: "2x2"},
		},
	}
	first, err := svc.SaveLayout(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("save layout first: %v", err)
	}
	second, err := svc.SaveLayout(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("save layout second: %v", err)
	}
	if first.LayoutID != second.LayoutID {
		t.Fatalf("expected idempotent replay to return same layout")
	}
}

func TestCreateViewRequiresIdempotency(t *testing.T) {
	svc := newService()
	_, err := svc.CreateCustomView(context.Background(), application.Actor{SubjectID: "user-1", Role: "creator"}, application.CreateCustomViewInput{
		ViewName:         "My View",
		WidgetIDs:        []string{"earnings"},
		DateRangeDefault: "30d",
	})
	if err != domain.ErrIdempotencyRequired {
		t.Fatalf("expected ErrIdempotencyRequired, got %v", err)
	}
}

func TestHandleInternalEventDedup(t *testing.T) {
	svc := newService()
	event := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "dashboard.cache_invalidation",
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.user_id",
		PartitionKey:     "user-1",
		SourceService:    "M55-Dashboard-Service",
		SchemaVersion:    "1.0",
		Data:             map[string]interface{}{"user_id": "user-1"},
	}
	if err := svc.HandleInternalEvent(context.Background(), event); err != nil {
		t.Fatalf("first handle event: %v", err)
	}
	if err := svc.HandleInternalEvent(context.Background(), event); err != nil {
		t.Fatalf("duplicate handle should no-op, got: %v", err)
	}
}
