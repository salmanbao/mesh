package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Traces:      repos.Traces,
		Spans:       repos.Spans,
		Tags:        repos.SpanTags,
		Policies:    repos.Policies,
		Exports:     repos.Exports,
		AuditLogs:   repos.AuditLogs,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
}

func validSpan() domain.IngestedSpan {
	start := time.Now().UTC().Add(-2 * time.Second)
	return domain.IngestedSpan{
		TraceID:       "0123456789abcdef0123456789abcdef",
		SpanID:        "0123456789abcdef",
		ServiceName:   "api-gateway",
		OperationName: "GET /v1/resource",
		StartTime:     start,
		EndTime:       start.Add(120 * time.Millisecond),
		Tags:          map[string]string{"http.method": "GET"},
		Environment:   "dev",
	}
}

func TestIngestSearchAndTraceDetail(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "dev-1", Role: "admin"}
	span := validSpan()
	_, err := svc.IngestSpans(context.Background(), actor, application.IngestInput{
		Format: "otlp",
		Spans:  []domain.IngestedSpan{span},
	})
	if err != nil {
		t.Fatalf("ingest spans: %v", err)
	}

	rows, err := svc.SearchTraces(context.Background(), actor, application.SearchInput{Limit: 10})
	if err != nil {
		t.Fatalf("search traces: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(rows))
	}

	detail, err := svc.GetTraceDetail(context.Background(), actor, span.TraceID)
	if err != nil {
		t.Fatalf("get trace detail: %v", err)
	}
	if detail.Trace.TraceID != span.TraceID {
		t.Fatalf("unexpected trace id: %s", detail.Trace.TraceID)
	}
	if len(detail.Spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(detail.Spans))
	}
}

func TestCreateSamplingPolicyIdempotentReplay(t *testing.T) {
	svc := newService()
	prob := 0.1
	actor := application.Actor{SubjectID: "sre-1", Role: "sre", IdempotencyKey: "idem-pol-1"}
	in := application.CreateSamplingPolicyInput{
		ServiceName: "payments",
		RuleType:    "probabilistic",
		Probability: &prob,
	}
	first, err := svc.CreateSamplingPolicy(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first create policy: %v", err)
	}
	second, err := svc.CreateSamplingPolicy(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second create policy: %v", err)
	}
	if first.PolicyID != second.PolicyID {
		t.Fatalf("expected idempotent replay to return same policy id")
	}
}

func TestCreateExportIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "dev-2", Role: "admin", IdempotencyKey: "idem-exp-1"}
	span := validSpan()
	first, err := svc.CreateExport(context.Background(), actor, application.CreateExportInput{
		TraceID: span.TraceID,
		Format:  "json",
	})
	if err != nil {
		t.Fatalf("first create export: %v", err)
	}
	second, err := svc.CreateExport(context.Background(), actor, application.CreateExportInput{
		TraceID: span.TraceID,
		Format:  "json",
	})
	if err != nil {
		t.Fatalf("second create export: %v", err)
	}
	if first.ExportID != second.ExportID {
		t.Fatalf("expected idempotent replay to return same export id")
	}
}

func TestHandleCanonicalEventUnsupportedDeduped(t *testing.T) {
	svc := newService()
	env := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "noncanonical.event",
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		SourceService:    "M00-Test",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		PartitionKeyPath: "data.id",
		PartitionKey:     "id-1",
		Data:             []byte(`{"id":"id-1"}`),
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != domain.ErrUnsupportedEventType {
		t.Fatalf("expected unsupported event, got %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != nil {
		t.Fatalf("expected duplicate event to no-op, got %v", err)
	}
}
