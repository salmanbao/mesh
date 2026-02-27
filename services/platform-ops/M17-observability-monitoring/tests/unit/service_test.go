package unit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Components:  repos.Components,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
}

func TestHealthDefaultsHealthy(t *testing.T) {
	svc := newService()
	out, err := svc.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	if out.Status != domain.StatusHealthy {
		t.Fatalf("expected healthy, got %s", out.Status)
	}
	for _, name := range []string{"database", "redis", "kafka"} {
		if _, ok := out.Checks[name]; !ok {
			t.Fatalf("missing %s check", name)
		}
	}
}

func TestUpsertComponentIdempotent(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-comp-1"}
	latency := 5000
	first, err := svc.UpsertComponent(context.Background(), actor, application.UpsertComponentInput{
		Name: "database", Status: "degraded", LatencyMS: &latency,
	})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	second, err := svc.UpsertComponent(context.Background(), actor, application.UpsertComponentInput{
		Name: "database", Status: "degraded", LatencyMS: &latency,
	})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if !first.LastChecked.Equal(second.LastChecked) {
		t.Fatalf("expected idempotent same response")
	}
}

func TestHealthUnhealthyWhenCriticalDegraded(t *testing.T) {
	svc := newService()
	lat := 5000
	_, err := svc.UpsertComponent(context.Background(), application.Actor{
		SubjectID: "admin-2", Role: "admin", IdempotencyKey: "idem-comp-2",
	}, application.UpsertComponentInput{Name: "database", Status: "degraded", LatencyMS: &lat})
	if err != nil {
		t.Fatalf("upsert component: %v", err)
	}
	out, err := svc.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	if out.Status != domain.StatusUnhealthy {
		t.Fatalf("expected unhealthy, got %s", out.Status)
	}
}

func TestMetricsRenderIncludesHttpSeries(t *testing.T) {
	svc := newService()
	svc.RecordHTTPMetric(context.Background(), application.MetricObservation{
		Method: "GET", Path: "/health", StatusCode: 200, Duration: 35 * time.Millisecond,
	})
	out, err := svc.RenderPrometheusMetrics(context.Background())
	if err != nil {
		t.Fatalf("render metrics: %v", err)
	}
	if !strings.Contains(out, "http_requests_total") {
		t.Fatalf("expected http_requests_total in metrics output")
	}
	if !strings.Contains(out, "http_request_duration_seconds_bucket") {
		t.Fatalf("expected histogram bucket in metrics output")
	}
}
