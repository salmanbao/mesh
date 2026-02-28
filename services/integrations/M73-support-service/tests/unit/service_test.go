package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/application"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Tickets:     repos.Tickets,
		Replies:     repos.Replies,
		CSAT:        repos.CSAT,
		Agents:      repos.Agents,
		Idempotency: repos.Idempotency,
	})
}

func TestCreateTicketIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "user-1", IdempotencyKey: "idem-ticket-create"}
	input := application.CreateTicketInput{
		Subject:     "Billing issue",
		Description: "I need help with a billing discrepancy on my payout.",
		Category:    "Billing",
	}
	first, err := svc.CreateTicket(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	second, err := svc.CreateTicket(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if first.TicketID != second.TicketID {
		t.Fatalf("expected same ticket id on replay: %s != %s", first.TicketID, second.TicketID)
	}
	if first.AssignedAgentID == "" {
		t.Fatalf("expected auto-assignment for billing ticket")
	}
}

func TestResolvedTicketReopensOnUserReply(t *testing.T) {
	svc := newService()
	created, err := svc.CreateTicket(context.Background(), application.Actor{SubjectID: "user-2", IdempotencyKey: "idem-create-2"}, application.CreateTicketInput{
		Subject:     "Technical issue",
		Description: "My upload is failing and I need assistance quickly.",
		Category:    "Technical",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	_, err = svc.UpdateTicket(context.Background(), application.Actor{SubjectID: "agent-technical", Role: "agent", IdempotencyKey: "idem-update"}, created.TicketID, application.UpdateTicketInput{Status: "resolved"})
	if err != nil {
		t.Fatalf("resolve ticket: %v", err)
	}
	_, err = svc.AddReply(context.Background(), application.Actor{SubjectID: "user-2", Role: "user", IdempotencyKey: "idem-reply"}, created.TicketID, application.AddReplyInput{ReplyType: "public", Body: "This still is not working."})
	if err != nil {
		t.Fatalf("user reply: %v", err)
	}
	updated, err := svc.GetTicket(context.Background(), application.Actor{SubjectID: "user-2"}, created.TicketID)
	if err != nil {
		t.Fatalf("get ticket: %v", err)
	}
	if updated.Status != "open" || updated.SubStatus != "new" {
		t.Fatalf("expected ticket reopened, got status=%s sub_status=%s", updated.Status, updated.SubStatus)
	}
}
