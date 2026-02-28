package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/application"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/domain"
)

func TestAuthorizeIntegrationAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Integrations: repos.Integrations,
		Credentials:  repos.Credentials,
		Workflows:    repos.Workflows,
		Executions:   repos.Executions,
		Webhooks:     repos.Webhooks,
		Deliveries:   repos.Deliveries,
		Analytics:    repos.Analytics,
		Logs:         repos.Logs,
		Idempotency:  repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "idem-int-1"}
	row, err := svc.AuthorizeIntegration(context.Background(), actor, application.AuthorizeIntegrationInput{
		IntegrationType: "Slack",
		IntegrationName: "my-slack",
	})
	if err != nil {
		t.Fatalf("authorize integration: %v", err)
	}
	if row.Status != domain.IntegrationStatusConnected {
		t.Fatalf("unexpected integration state: %+v", row)
	}
	replay, err := svc.AuthorizeIntegration(context.Background(), actor, application.AuthorizeIntegrationInput{
		IntegrationType: "Slack",
		IntegrationName: "my-slack",
	})
	if err != nil {
		t.Fatalf("authorize replay: %v", err)
	}
	if replay.IntegrationID != row.IntegrationID {
		t.Fatalf("expected replay to reuse integration id")
	}
}

func TestWorkflowPublishAndTest(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Integrations: repos.Integrations,
		Credentials:  repos.Credentials,
		Workflows:    repos.Workflows,
		Executions:   repos.Executions,
		Webhooks:     repos.Webhooks,
		Deliveries:   repos.Deliveries,
		Analytics:    repos.Analytics,
		Logs:         repos.Logs,
		Idempotency:  repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "user-2", Role: "user", IdempotencyKey: "idem-int-2"}
	intg, err := svc.AuthorizeIntegration(context.Background(), actor, application.AuthorizeIntegrationInput{IntegrationType: "Slack"})
	if err != nil {
		t.Fatalf("authorize integration: %v", err)
	}
	workflow, err := svc.CreateWorkflow(context.Background(), application.Actor{SubjectID: "user-2", Role: "user", IdempotencyKey: "idem-wf-1"}, application.CreateWorkflowInput{
		WorkflowName:     "Notify Slack",
		TriggerEventType: "submission.approved",
		ActionType:       "send_slack_message",
		IntegrationID:    intg.IntegrationID,
	})
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if workflow.Status != domain.WorkflowStatusDraft {
		t.Fatalf("unexpected workflow: %+v", workflow)
	}
	published, err := svc.PublishWorkflow(context.Background(), application.Actor{SubjectID: "user-2", Role: "user", IdempotencyKey: "idem-pub-1"}, workflow.WorkflowID)
	if err != nil {
		t.Fatalf("publish workflow: %v", err)
	}
	if published.Status != domain.WorkflowStatusPublished {
		t.Fatalf("unexpected published workflow: %+v", published)
	}
	exec, err := svc.TestWorkflow(context.Background(), application.Actor{SubjectID: "user-2", Role: "user"}, workflow.WorkflowID)
	if err != nil {
		t.Fatalf("test workflow: %v", err)
	}
	if exec.Status != domain.ExecutionStatusSuccess || !exec.TestRun {
		t.Fatalf("unexpected execution: %+v", exec)
	}
}

func TestWebhookAndChatPostMessage(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Integrations: repos.Integrations,
		Credentials:  repos.Credentials,
		Workflows:    repos.Workflows,
		Executions:   repos.Executions,
		Webhooks:     repos.Webhooks,
		Deliveries:   repos.Deliveries,
		Analytics:    repos.Analytics,
		Logs:         repos.Logs,
		Idempotency:  repos.Idempotency,
	})
	webhook, err := svc.CreateWebhook(context.Background(), application.Actor{SubjectID: "user-3", Role: "user", IdempotencyKey: "idem-wh-1"}, application.CreateWebhookInput{
		EndpointURL: "https://example.com/hook",
		EventType:   "submission.created",
	})
	if err != nil {
		t.Fatalf("create webhook: %v", err)
	}
	delivery, err := svc.TestWebhook(context.Background(), application.Actor{SubjectID: "user-3", Role: "user"}, webhook.WebhookID)
	if err != nil {
		t.Fatalf("test webhook: %v", err)
	}
	if !delivery.TestEvent || delivery.Status != domain.ExecutionStatusSuccess {
		t.Fatalf("unexpected delivery: %+v", delivery)
	}
	channel, messageID, err := svc.ChatPostMessage(context.Background(), application.Actor{SubjectID: "user-3", Role: "user"}, "#alerts")
	if err != nil {
		t.Fatalf("chat post message: %v", err)
	}
	if channel != "#alerts" || messageID == "" {
		t.Fatalf("unexpected message output")
	}
}
