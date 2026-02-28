package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Configs:      repos.Configs,
		Purges:       repos.Purges,
		Metrics:      repos.Metrics,
		Certificates: repos.Certificates,
		Idempotency:  repos.Idempotency,
	})
}

func TestCreateConfigIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "ops-1", Role: "ops_admin", IdempotencyKey: "idem-config"}
	input := application.CreateConfigInput{Provider: "cloudflare", Config: map[string]any{"cache_ttl": 300}}
	first, err := svc.CreateConfig(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}
	second, err := svc.CreateConfig(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("replay config: %v", err)
	}
	if first.ConfigID != second.ConfigID || first.Version != 1 || second.Version != 1 {
		t.Fatalf("unexpected idempotent replay result: first=%+v second=%+v", first, second)
	}
}

func TestPurgeRequiresAdmin(t *testing.T) {
	svc := newService()
	_, err := svc.Purge(context.Background(), application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "idem-purge"}, application.PurgeInput{Scope: "url", Target: "https://cdn.example.com/a.jpg"})
	if err == nil {
		t.Fatalf("expected purge to reject non-admin actor")
	}
}
