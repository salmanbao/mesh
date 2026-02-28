package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{Licenses: repos.Licenses, Activations: repos.Activations, Revocations: repos.Revocations, Configs: repos.Configs, Idempotency: repos.Idempotency})
}

func TestActivateIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "user-1", IdempotencyKey: "idem-activate", ClientIP: "203.0.113.10"}
	input := application.ActivateInput{LicenseKey: "ABCDE-FGHIJ-KLMNO-PQRST", DeviceID: "device-1", DeviceFingerprint: "fp-1"}
	first, err := svc.Activate(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	second, err := svc.Activate(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("activate replay: %v", err)
	}
	if first["device_id"] != second["device_id"] || first["license_id"] != second["license_id"] {
		t.Fatalf("unexpected replay result: first=%v second=%v", first, second)
	}
}

func TestValidationRateLimit(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "user-1", ClientIP: "203.0.113.20"}
	for i := 0; i < 5; i++ {
		if _, err := svc.Validate(context.Background(), actor, "ABCDE-FGHIJ-KLMNO-PQRST"); err != nil {
			t.Fatalf("unexpected validate error on attempt %d: %v", i, err)
		}
	}
	if _, err := svc.Validate(context.Background(), actor, "ABCDE-FGHIJ-KLMNO-PQRST"); err == nil {
		t.Fatalf("expected rate limit on sixth validation")
	}
}
