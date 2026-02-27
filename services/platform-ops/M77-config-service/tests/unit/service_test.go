package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Keys:        repos.Keys,
		Values:      repos.Values,
		Versions:    repos.Versions,
		Rules:       repos.Rules,
		Audits:      repos.Audits,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
}

func TestPatchConfigIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-cfg-1"}
	in := application.PatchConfigInput{
		Key:          "payout.instant_fee_percent",
		Environment:  domain.EnvProduction,
		ServiceScope: "reward-engine",
		ValueType:    domain.ValueTypeNumber,
		Value:        10,
	}
	first, err := svc.PatchConfig(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first patch: %v", err)
	}
	second, err := svc.PatchConfig(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second patch: %v", err)
	}
	if first.Version != second.Version {
		t.Fatalf("expected idempotent replay same version, got %d and %d", first.Version, second.Version)
	}
}

func TestGetConfigFallbackAndEncryptedMasking(t *testing.T) {
	svc := newService()
	admin := application.Actor{SubjectID: "admin-1", Role: "admin"}
	mustPatch := func(key, scope, valueType string, value any, idem string) {
		admin.IdempotencyKey = idem
		if _, err := svc.PatchConfig(context.Background(), admin, application.PatchConfigInput{
			Key:          key,
			Environment:  domain.EnvProduction,
			ServiceScope: scope,
			ValueType:    valueType,
			Value:        value,
		}); err != nil {
			t.Fatalf("patch %s: %v", key, err)
		}
	}
	mustPatch("campaign.min_rate_per_1k", domain.GlobalServiceScope, domain.ValueTypeNumber, 0.1, "idem-1")
	mustPatch("payout.instant_fee_percent", "reward-engine", domain.ValueTypeNumber, 15, "idem-2")
	mustPatch("stripe.secret_key", domain.GlobalServiceScope, domain.ValueTypeEncrypted, "sk_live_123", "idem-3")

	out, err := svc.GetConfig(context.Background(), application.Actor{SubjectID: "svc-reward", Role: "service"}, application.GetConfigInput{
		Environment:  domain.EnvProduction,
		ServiceScope: "reward-engine",
	})
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if out["stripe.secret_key"] != "***" {
		t.Fatalf("expected encrypted value masked, got %#v", out["stripe.secret_key"])
	}
	if v, ok := out["payout.instant_fee_percent"].(float64); !ok || v != 15 {
		t.Fatalf("expected service-scoped override 15, got %#v", out["payout.instant_fee_percent"])
	}
	if v, ok := out["campaign.min_rate_per_1k"].(float64); !ok || v != 0.1 {
		t.Fatalf("expected global fallback 0.1, got %#v", out["campaign.min_rate_per_1k"])
	}
}

func TestRolloutRuleDisablesBooleanForNonMatchingCohort(t *testing.T) {
	svc := newService()
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-flag-1"}
	if _, err := svc.PatchConfig(context.Background(), admin, application.PatchConfigInput{
		Key:          "ai_clipping_tool.enabled",
		Environment:  domain.EnvProduction,
		ServiceScope: domain.GlobalServiceScope,
		ValueType:    domain.ValueTypeBoolean,
		Value:        true,
	}); err != nil {
		t.Fatalf("patch flag: %v", err)
	}
	admin.IdempotencyKey = "idem-rule-1"
	if _, err := svc.CreateRolloutRule(context.Background(), admin, application.CreateRolloutRuleInput{
		Key:        "ai_clipping_tool.enabled",
		RuleType:   domain.RuleTypePercentage,
		Percentage: 0,
	}); err != nil {
		t.Fatalf("create rollout rule: %v", err)
	}
	out, err := svc.GetConfig(context.Background(), application.Actor{SubjectID: "sdk-1", Role: "service"}, application.GetConfigInput{
		Environment:  domain.EnvProduction,
		ServiceScope: domain.GlobalServiceScope,
		UserID:       "user-123",
	})
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if v, ok := out["ai_clipping_tool.enabled"].(bool); !ok || v {
		t.Fatalf("expected rollout gated false, got %#v", out["ai_clipping_tool.enabled"])
	}
}

func TestRollbackRestoresPreviousVersion(t *testing.T) {
	svc := newService()
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-rb-1"}
	if _, err := svc.PatchConfig(context.Background(), admin, application.PatchConfigInput{
		Key:          "limits.max_upload_mb",
		Environment:  domain.EnvProduction,
		ServiceScope: domain.GlobalServiceScope,
		ValueType:    domain.ValueTypeNumber,
		Value:        50,
	}); err != nil {
		t.Fatalf("patch v1: %v", err)
	}
	admin.IdempotencyKey = "idem-rb-2"
	if _, err := svc.PatchConfig(context.Background(), admin, application.PatchConfigInput{
		Key:          "limits.max_upload_mb",
		Environment:  domain.EnvProduction,
		ServiceScope: domain.GlobalServiceScope,
		ValueType:    domain.ValueTypeNumber,
		Value:        75,
	}); err != nil {
		t.Fatalf("patch v2: %v", err)
	}
	admin.IdempotencyKey = "idem-rb-3"
	out, err := svc.RollbackConfig(context.Background(), admin, application.RollbackConfigInput{
		Key:          "limits.max_upload_mb",
		Environment:  domain.EnvProduction,
		ServiceScope: domain.GlobalServiceScope,
		Version:      1,
	})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if out.RolledBackTo != 1 || out.Version != 3 {
		t.Fatalf("unexpected rollback result: %+v", out)
	}
	cfg, err := svc.GetConfig(context.Background(), application.Actor{SubjectID: "svc", Role: "service"}, application.GetConfigInput{
		Environment:  domain.EnvProduction,
		ServiceScope: domain.GlobalServiceScope,
	})
	if err != nil {
		t.Fatalf("get config after rollback: %v", err)
	}
	if v, ok := cfg["limits.max_upload_mb"].(float64); !ok || v != 50 {
		t.Fatalf("expected rollback value 50, got %#v", cfg["limits.max_upload_mb"])
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
