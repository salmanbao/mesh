package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M84-data-migration-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{Plans: repos.Plans, Runs: repos.Runs, Registry: repos.Registry, Backfills: repos.Backfills, Metrics: repos.Metrics, Idempotency: repos.Idempotency})
}

func TestCreatePlanIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "ops-1", Role: "migration_operator", IdempotencyKey: "idem-plan"}
	input := application.CreatePlanInput{ServiceName: "M01-Authentication-Service", Environment: "staging", Version: "2026.03.01", Plan: map[string]any{"steps": 3}}
	first, err := svc.CreatePlan(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	second, err := svc.CreatePlan(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("replay plan: %v", err)
	}
	if first.PlanID != second.PlanID {
		t.Fatalf("expected same plan id on replay: %s != %s", first.PlanID, second.PlanID)
	}
}

func TestCreateRunRequiresMFA(t *testing.T) {
	svc := newService()
	plan, err := svc.CreatePlan(context.Background(), application.Actor{SubjectID: "ops-1", Role: "migration_operator", IdempotencyKey: "idem-plan-2"}, application.CreatePlanInput{ServiceName: "M10-Social-Integration-Verification-Service", Environment: "staging", Version: "2026.03.02", Plan: map[string]any{"ddl": "add column"}})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	_, err = svc.CreateRun(context.Background(), application.Actor{SubjectID: "ops-1", Role: "migration_operator", IdempotencyKey: "idem-run", MFAVerified: false}, application.CreateRunInput{PlanID: plan.PlanID})
	if err == nil {
		t.Fatalf("expected run creation to require MFA")
	}
}
