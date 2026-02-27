package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/domain"
)

func newService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Rules:       repos.Rules,
		Alerts:      repos.Alerts,
		Incidents:   repos.Incidents,
		Silences:    repos.Silences,
		Dashboards:  repos.Dashboards,
		Audits:      repos.Audits,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
	return svc, repos
}

func TestCreateAlertRuleIdempotentReplay(t *testing.T) {
	svc, _ := newService()
	actor := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-rule-1"}
	in := application.CreateAlertRuleInput{
		Name:            "auth-service error rate",
		Query:           "error_rate{service='auth-service'}",
		Threshold:       0.01,
		DurationSeconds: 300,
		Severity:        "critical",
		Enabled:         true,
		Service:         "auth-service",
	}
	first, err := svc.CreateAlertRule(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first create rule: %v", err)
	}
	second, err := svc.CreateAlertRule(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second create rule: %v", err)
	}
	if first.RuleID != second.RuleID {
		t.Fatalf("expected idempotent replay with same rule id")
	}
}

func TestCreateSilenceRequiresAdmin(t *testing.T) {
	svc, _ := newService()
	now := time.Now().UTC()
	_, err := svc.CreateSilence(context.Background(), application.Actor{
		SubjectID:      "dev-1",
		Role:           "developer",
		IdempotencyKey: "idem-silence-1",
	}, application.CreateSilenceInput{
		RuleID:  "rule-1",
		Reason:  "maintenance",
		StartAt: now,
		EndAt:   now.Add(time.Hour),
	})
	if err != domain.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestListIncidentsByStatus(t *testing.T) {
	svc, repos := newService()
	now := time.Now().UTC()
	_ = repos.Incidents.Create(context.Background(), domain.Incident{
		IncidentID: "inc-1",
		AlertID:    "alert-1",
		Service:    "auth-service",
		Severity:   "critical",
		Status:     domain.IncidentStatusInvestigating,
		CreatedAt:  now,
	})
	_ = repos.Incidents.Create(context.Background(), domain.Incident{
		IncidentID: "inc-2",
		AlertID:    "alert-2",
		Service:    "payments",
		Severity:   "warning",
		Status:     domain.IncidentStatusResolved,
		CreatedAt:  now,
	})
	rows, err := svc.ListIncidents(context.Background(), application.Actor{SubjectID: "dev-1", Role: "developer"}, application.ListIncidentsInput{
		Status: domain.IncidentStatusInvestigating,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(rows) != 1 || rows[0].IncidentID != "inc-1" {
		t.Fatalf("expected only inc-1, got %#v", rows)
	}
}

func TestHandleCanonicalEventUnsupportedDeduped(t *testing.T) {
	svc, _ := newService()
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
