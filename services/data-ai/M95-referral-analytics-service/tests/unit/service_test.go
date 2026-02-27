package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/domain"
)

func newTestService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Warehouse:    repos.Warehouse,
		Exports:      repos.Exports,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		Affiliate:    grpcadapter.NewAffiliateClient("stub"),
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})
	return svc, repos
}

func TestGetFunnel(t *testing.T) {
	svc, _ := newTestService()
	out, err := svc.GetFunnel(context.Background(), application.Actor{SubjectID: "user_1", Role: "analyst"}, application.DateRangeInput{})
	if err != nil {
		t.Fatalf("GetFunnel error: %v", err)
	}
	if out.Clicks <= 0 || out.Signups <= 0 {
		t.Fatalf("expected seeded funnel metrics, got %+v", out)
	}
}

func TestRequestExportIdempotency(t *testing.T) {
	svc, _ := newTestService()
	actor := application.Actor{SubjectID: "analyst_1", Role: "analyst", IdempotencyKey: "idem-export-1"}
	in := application.ExportInput{ExportType: "leaderboard", Period: "30d", Format: "csv", Filters: map[string]string{"platform": "instagram"}}
	first, err := svc.RequestExport(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first export error: %v", err)
	}
	second, err := svc.RequestExport(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second export error: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected idempotent replay same id, got %s vs %s", first.ID, second.ID)
	}
	_, err = svc.RequestExport(context.Background(), actor, application.ExportInput{ExportType: "geo", Period: "30d", Format: "csv"})
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestHandleCanonicalEventUnsupportedButValidatesEnvelope(t *testing.T) {
	svc, _ := newTestService()
	payload, _ := json.Marshal(map[string]any{"affiliate_id": "a1"})
	err := svc.HandleCanonicalEvent(context.Background(), contracts.EventEnvelope{
		EventID: "evt1", EventType: "affiliate.click.tracked", EventClass: domain.CanonicalEventClassDomain,
		OccurredAt: time.Now().UTC(), PartitionKeyPath: "data.affiliate_id", PartitionKey: "a1",
		SourceService: "M89-Affiliate-Service", TraceID: "trace1", SchemaVersion: "v1", Data: payload,
	})
	if err != domain.ErrUnsupportedEventType {
		t.Fatalf("expected unsupported event type, got %v", err)
	}
}
