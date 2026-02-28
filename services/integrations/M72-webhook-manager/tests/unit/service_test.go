package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Analytics:   repos.Analytics,
		Idempotency: repos.Idempotency,
	})
}

func TestCreateWebhookIdempotencyReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "user-1", IdempotencyKey: "idem-create"}
	input := application.CreateWebhookInput{
		EndpointURL: "https://example.com/webhook",
		EventTypes:  []string{"submission.created", "submission.created"},
	}

	first, err := svc.CreateWebhook(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}
	second, err := svc.CreateWebhook(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if first.WebhookID != second.WebhookID {
		t.Fatalf("expected idempotent replay to return same webhook id: first=%s second=%s", first.WebhookID, second.WebhookID)
	}
	if len(second.EventTypes) != 1 {
		t.Fatalf("expected deduped event types, got=%v", second.EventTypes)
	}
}

func TestUpdateDeleteLifecycle(t *testing.T) {
	svc := newService()
	createActor := application.Actor{SubjectID: "user-1", IdempotencyKey: "idem-create-2"}
	created, err := svc.CreateWebhook(context.Background(), createActor, application.CreateWebhookInput{
		EndpointURL: "https://example.com/hooks",
		EventTypes:  []string{"campaign.created"},
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}

	disable := false
	updated, err := svc.UpdateWebhook(context.Background(), application.Actor{SubjectID: "user-1", IdempotencyKey: "idem-update"}, created.WebhookID, application.UpdateWebhookInput{
		BatchModeEnabled:   &disable,
		RateLimitPerMinute: 50,
		Status:             "disabled",
	})
	if err != nil {
		t.Fatalf("update webhook: %v", err)
	}
	if updated.Status != "disabled" || updated.RateLimitPerMinute != 50 {
		t.Fatalf("unexpected update result: %+v", updated)
	}

	deleted, err := svc.DeleteWebhook(context.Background(), application.Actor{SubjectID: "user-1"}, created.WebhookID)
	if err != nil {
		t.Fatalf("delete webhook: %v", err)
	}
	if deleted.Status != "deleted" || deleted.DeletedAt.IsZero() {
		t.Fatalf("expected soft deleted webhook, got=%+v", deleted)
	}

	repeatDelete, err := svc.DeleteWebhook(context.Background(), application.Actor{SubjectID: "user-1"}, created.WebhookID)
	if err != nil {
		t.Fatalf("repeat delete: %v", err)
	}
	if repeatDelete.Status != "deleted" {
		t.Fatalf("expected repeat delete to remain deleted, got=%+v", repeatDelete)
	}
}
