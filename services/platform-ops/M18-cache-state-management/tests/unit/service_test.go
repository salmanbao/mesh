package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/domain"
)

func newService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:    "M18-Cache-State-Management",
			Version:        "test",
			DefaultTTL:     10 * time.Second,
			IdempotencyTTL: 7 * 24 * time.Hour,
			EventDedupTTL:  7 * 24 * time.Hour,
		},
		Cache:       repos.Cache,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
	return svc, repos
}

func TestPutCacheIdempotency(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	actor := application.Actor{SubjectID: "svc-a", Role: "service", IdempotencyKey: "idem-1", RequestID: "req-1"}
	payload := json.RawMessage(`{"k":"v"}`)

	first, err := svc.PutCache(ctx, actor, "demo:key", payload, 60)
	if err != nil {
		t.Fatalf("put cache failed: %v", err)
	}
	second, err := svc.PutCache(ctx, actor, "demo:key", payload, 60)
	if err != nil {
		t.Fatalf("idempotent put failed: %v", err)
	}
	if string(first.Value) != string(second.Value) || first.Key != second.Key {
		t.Fatalf("expected same cached response")
	}

	_, err = svc.PutCache(ctx, actor, "demo:key", json.RawMessage(`{"k":"other"}`), 60)
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	actor := application.Actor{SubjectID: "svc-a", Role: "service", IdempotencyKey: "idem-2", RequestID: "req-2"}
	_, err := svc.PutCache(ctx, actor, "ttl:key", json.RawMessage(`{"v":1}`), 1)
	if err != nil {
		t.Fatalf("put cache failed: %v", err)
	}

	readActor := application.Actor{SubjectID: "svc-a", Role: "service", RequestID: "req-3"}
	item, err := svc.GetCache(ctx, readActor, "ttl:key")
	if err != nil || !item.Found {
		t.Fatalf("expected found item, err=%v", err)
	}
	if item.TTLSeconds <= 0 {
		t.Fatalf("expected positive ttl")
	}

	time.Sleep(1100 * time.Millisecond)
	item, err = svc.GetCache(ctx, readActor, "ttl:key")
	if err != nil {
		t.Fatalf("get expired item failed: %v", err)
	}
	if item.Found {
		t.Fatalf("expected expired item to be missing")
	}
}

func TestHandleCanonicalEventUnsupported(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	env := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "cache.invalidate",
		EventClass:       "domain",
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.key",
		PartitionKey:     "demo:key",
		SourceService:    "M18-Cache-State-Management",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		Data:             json.RawMessage(`{"key":"demo:key"}`),
	}
	if err := svc.HandleCanonicalEvent(ctx, env); err != domain.ErrUnsupportedEventType {
		t.Fatalf("expected unsupported_event_type, got %v", err)
	}
	if err := svc.HandleCanonicalEvent(ctx, env); err != nil {
		t.Fatalf("expected duplicate no-op, got %v", err)
	}
}

func TestMetricsSnapshot(t *testing.T) {
	svc, _ := newService()
	ctx := context.Background()
	mutatingActor := application.Actor{SubjectID: "svc-a", Role: "service", IdempotencyKey: "idem-4", RequestID: "req-4"}
	readActor := application.Actor{SubjectID: "svc-a", Role: "service", RequestID: "req-5"}
	_, _ = svc.PutCache(ctx, mutatingActor, "metrics:key", json.RawMessage(`{"p":1}`), 60)
	_, _ = svc.GetCache(ctx, readActor, "metrics:key")
	_, _ = svc.GetCache(ctx, readActor, "metrics:missing")

	m, err := svc.GetMetrics(ctx, readActor)
	if err != nil {
		t.Fatalf("metrics failed: %v", err)
	}
	if m.Hits < 1 || m.Misses < 1 {
		t.Fatalf("expected hits and misses to be tracked, got hits=%d misses=%d", m.Hits, m.Misses)
	}
	if m.MemoryUsedBytes <= 0 {
		t.Fatalf("expected memory_used_bytes > 0")
	}
}
