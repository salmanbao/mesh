package unit

import (
	"context"
	"testing"

	eventadapter "github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/adapters/events"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/application"
)

func newService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Accounts: repos.Accounts, Metrics: repos.Metrics, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup, Outbox: repos.Outbox,
		DomainEvents: eventadapter.NewMemoryDomainPublisher(), Analytics: eventadapter.NewMemoryAnalyticsPublisher(), DLQ: eventadapter.NewLoggingDLQPublisher(),
	})
	return svc, repos
}

func TestOAuthCallbackEnqueuesCanonicalEvents(t *testing.T) {
	svc, repos := newService()
	actor := application.Actor{SubjectID: "user_1", Role: "user", RequestID: "req_1", IdempotencyKey: "idem-callback-1"}
	_, err := svc.OAuthCallback(context.Background(), actor, application.CallbackInput{Provider: "instagram", UserID: "user_1", Code: "oauth-code", State: "state-1", Handle: "creator"})
	if err != nil {
		t.Fatalf("OAuthCallback: %v", err)
	}
	pending, err := repos.Outbox.ListPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if got := len(pending); got != 2 {
		t.Fatalf("expected 2 outbox events (connected + status_changed), got %d", got)
	}
}

func TestConnectStartIdempotentReplay(t *testing.T) {
	svc, _ := newService()
	actor := application.Actor{SubjectID: "user_1", Role: "user", RequestID: "req_1", IdempotencyKey: "idem-connect-1"}
	first, err := svc.ConnectStart(context.Background(), actor, application.ConnectInput{Provider: "instagram", UserID: "user_1"})
	if err != nil {
		t.Fatalf("first ConnectStart: %v", err)
	}
	second, err := svc.ConnectStart(context.Background(), actor, application.ConnectInput{Provider: "instagram", UserID: "user_1"})
	if err != nil {
		t.Fatalf("second ConnectStart: %v", err)
	}
	if first.State != second.State || first.AuthURL != second.AuthURL {
		t.Fatalf("expected idempotent replay to match; first=%+v second=%+v", first, second)
	}
}
