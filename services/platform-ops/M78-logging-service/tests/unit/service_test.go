package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Logs:        repos.Logs,
		Alerts:      repos.Alerts,
		Exports:     repos.Exports,
		Audits:      repos.Audits,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
}

func TestIngestLogsIdempotentReplayAndRedaction(t *testing.T) {
	svc := newService()
	actor := application.Actor{
		SubjectID:      "svc-ingester",
		Role:           "system",
		IdempotencyKey: "idem-log-1",
	}
	ts := time.Now().UTC()
	in := application.IngestLogsInput{
		Logs: []application.IngestLogRecordInput{
			{
				Timestamp: ts,
				Level:     "error",
				Service:   "payments",
				Message:   "token=abc123 email=alice@example.com user_id=42",
			},
		},
	}

	first, err := svc.IngestLogs(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := svc.IngestLogs(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if first.Ingested != 1 || second.Ingested != 1 {
		t.Fatalf("unexpected ingest counts: %+v %+v", first, second)
	}

	rows, err := svc.SearchLogs(context.Background(), application.Actor{SubjectID: "aud-1", Role: "auditor"}, application.SearchLogsInput{
		Service: "payments",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("search logs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 log row, got %d", len(rows))
	}
	if !rows[0].Redacted {
		t.Fatalf("expected redacted row")
	}
	if rows[0].Message == in.Logs[0].Message {
		t.Fatalf("expected message to be redacted")
	}
}

func TestCreateAlertRuleRequiresAdmin(t *testing.T) {
	svc := newService()
	_, err := svc.CreateAlertRule(context.Background(), application.Actor{
		SubjectID:      "dev-1",
		Role:           "developer",
		IdempotencyKey: "idem-rule-1",
	}, application.CreateAlertRuleInput{
		Service:   "payments",
		Severity:  "critical",
		Enabled:   true,
		Condition: map[string]any{"error_rate_gt": 0.1},
	})
	if err != domain.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
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
		t.Fatalf("expected duplicate no-op, got %v", err)
	}
}
