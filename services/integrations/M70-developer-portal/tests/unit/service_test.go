package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/application"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/domain"
)

func TestRegisterAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Developers:  repos.Developers,
		Sessions:    repos.Sessions,
		APIKeys:     repos.APIKeys,
		Rotations:   repos.Rotations,
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Usage:       repos.Usage,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "requester-1", Role: "developer", IdempotencyKey: "idem-register-1"}
	dev, sess, err := svc.RegisterDeveloper(context.Background(), actor, application.RegisterDeveloperInput{
		Email:   "dev@example.com",
		AppName: "Portal App",
	})
	if err != nil {
		t.Fatalf("register developer: %v", err)
	}
	if dev.Status != domain.DeveloperStatusActive || sess.DeveloperID != dev.DeveloperID {
		t.Fatalf("unexpected registration output: dev=%+v sess=%+v", dev, sess)
	}
	replayDev, replaySess, err := svc.RegisterDeveloper(context.Background(), actor, application.RegisterDeveloperInput{
		Email:   "dev@example.com",
		AppName: "Portal App",
	})
	if err != nil {
		t.Fatalf("register replay: %v", err)
	}
	if replayDev.DeveloperID != dev.DeveloperID || replaySess.SessionID != sess.SessionID {
		t.Fatalf("expected replay ids to match")
	}
}

func TestAPIKeyLifecycleAndWebhook(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Developers:  repos.Developers,
		Sessions:    repos.Sessions,
		APIKeys:     repos.APIKeys,
		Rotations:   repos.Rotations,
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Usage:       repos.Usage,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-register-2"}
	dev, _, err := svc.RegisterDeveloper(context.Background(), admin, application.RegisterDeveloperInput{
		Email:   "dev2@example.com",
		AppName: "Portal App 2",
	})
	if err != nil {
		t.Fatalf("register developer: %v", err)
	}

	key, err := svc.CreateAPIKey(context.Background(), application.Actor{SubjectID: dev.DeveloperID, Role: "developer", IdempotencyKey: "idem-key-1"}, application.CreateAPIKeyInput{
		Label: "Primary",
	})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if key.Status != domain.APIKeyStatusActive {
		t.Fatalf("unexpected key: %+v", key)
	}

	rotation, oldKey, newKey, err := svc.RotateAPIKey(context.Background(), application.Actor{SubjectID: dev.DeveloperID, Role: "developer", IdempotencyKey: "idem-rotate-1"}, key.KeyID)
	if err != nil {
		t.Fatalf("rotate key: %v", err)
	}
	if rotation.RotationID == "" || oldKey.Status != domain.APIKeyStatusDeprecated || newKey.Status != domain.APIKeyStatusActive {
		t.Fatalf("unexpected rotation output")
	}

	revoked, err := svc.RevokeAPIKey(context.Background(), application.Actor{SubjectID: dev.DeveloperID, Role: "developer"}, newKey.KeyID)
	if err != nil {
		t.Fatalf("revoke key: %v", err)
	}
	if revoked.Status != domain.APIKeyStatusRevoked {
		t.Fatalf("unexpected revoked key: %+v", revoked)
	}

	webhook, err := svc.CreateWebhook(context.Background(), application.Actor{SubjectID: dev.DeveloperID, Role: "developer", IdempotencyKey: "idem-webhook-1"}, application.CreateWebhookInput{
		URL:       "https://example.com/hook",
		EventType: "order.created",
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}
	delivery, err := svc.TestWebhook(context.Background(), application.Actor{SubjectID: dev.DeveloperID, Role: "developer"}, webhook.WebhookID)
	if err != nil {
		t.Fatalf("test webhook: %v", err)
	}
	if !delivery.TestEvent || delivery.Status != domain.DeliveryStatusSuccess {
		t.Fatalf("unexpected delivery: %+v", delivery)
	}
}
