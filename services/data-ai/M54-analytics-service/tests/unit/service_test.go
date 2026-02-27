package unit

import (
	"context"
	"testing"
	"time"

	eventsadapter "github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:    "M54-Analytics-Service",
			IdempotencyTTL: 7 * 24 * time.Hour,
			EventDedupTTL:  7 * 24 * time.Hour,
		},
		Warehouse:   repos.Warehouse,
		Exports:     repos.Exports,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Voting:      grpcadapter.NewVotingClient(""),
		Social:      grpcadapter.NewSocialClient(""),
		Tracking:    grpcadapter.NewTrackingClient(""),
		Submission:  grpcadapter.NewSubmissionClient(""),
		Finance:     grpcadapter.NewFinanceClient(""),
		DLQ:         eventsadapter.NewLoggingDLQPublisher(),
	})
}

func TestRequestExportIdempotency(t *testing.T) {
	t.Parallel()

	svc := newService()
	actor := application.Actor{
		SubjectID:      "creator-1",
		Role:           "creator",
		RequestID:      "req-1",
		IdempotencyKey: "export:req:creator-1:day-1",
	}
	input := application.ExportInput{
		ReportType: "creator_dashboard",
		Format:     "csv",
		DateFrom:   "2026-02-01",
		DateTo:     "2026-02-26",
		Filters:    map[string]string{"platform": "tiktok"},
	}
	first, err := svc.RequestExport(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first export request: %v", err)
	}
	second, err := svc.RequestExport(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second export request: %v", err)
	}
	if first.ExportID != second.ExportID {
		t.Fatalf("expected same export id for idempotent replay")
	}
}

func TestHandleCanonicalEventDedup(t *testing.T) {
	t.Parallel()

	svc := newService()
	event := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        domain.EventTransactionSucceeded,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.transaction_id",
		PartitionKey:     "txn-1",
		SourceService:    "M39-Finance-Service",
		TraceID:          "trace-1",
		SchemaVersion:    "1.0",
		Data: []byte(`{
			"transaction_id":"txn-1",
			"user_id":"creator-1",
			"amount":75.50,
			"occurred_at":"2026-02-26T00:00:00Z"
		}`),
	}
	if err := svc.HandleCanonicalEvent(context.Background(), event); err != nil {
		t.Fatalf("first handle event: %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), event); err != nil {
		t.Fatalf("duplicate handle should no-op, got: %v", err)
	}
}

func TestFinancialReportForbiddenForCreator(t *testing.T) {
	t.Parallel()

	svc := newService()
	_, err := svc.GetAdminFinancialReport(context.Background(), application.Actor{
		SubjectID: "creator-1",
		Role:      "creator",
	}, application.FinancialReportInput{})
	if err != domain.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
