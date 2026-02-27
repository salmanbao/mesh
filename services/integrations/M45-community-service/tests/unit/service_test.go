package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{Integrations: repos.Integrations, Mappings: repos.Mappings, Grants: repos.Grants, AuditLogs: repos.AuditLogs, HealthChecks: repos.HealthChecks, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup})
}

func TestConnectIntegrationIdempotent(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "creator-1", Role: "creator", IdempotencyKey: "idem-connect-1"}
	in := application.ConnectIntegrationInput{Platform: "discord", CommunityName: "Test Guild", Config: map[string]string{"server_id": "123", "oauth_code": "abc"}}
	first, err := svc.ConnectIntegration(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first connect: %v", err)
	}
	second, err := svc.ConnectIntegration(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second connect: %v", err)
	}
	if first.IntegrationID != second.IntegrationID {
		t.Fatalf("expected idempotent integration id")
	}
}

func TestManualGrantCreatesGrantAndAudit(t *testing.T) {
	svc := newService()
	creator := application.Actor{SubjectID: "creator-1", Role: "creator", IdempotencyKey: "idem-connect-2"}
	integration, err := svc.ConnectIntegration(context.Background(), creator, application.ConnectIntegrationInput{Platform: "discord", CommunityName: "Guild", Config: map[string]string{"server_id": "123"}})
	if err != nil {
		t.Fatalf("connect integration: %v", err)
	}
	admin := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-grant-1"}
	grant, err := svc.CreateManualGrant(context.Background(), admin, application.ManualGrantInput{UserID: "user-1", ProductID: "prod-1", IntegrationID: integration.IntegrationID, Reason: "testing"})
	if err != nil {
		t.Fatalf("manual grant: %v", err)
	}
	if grant.GrantID == "" || string(grant.Status) != "active" {
		t.Fatalf("unexpected grant result: %+v", grant)
	}
	logs, err := svc.ListAuditLogs(context.Background(), application.Actor{SubjectID: "support-1", Role: "support"}, "", nil, nil)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected audit logs for connect + grant")
	}
}
