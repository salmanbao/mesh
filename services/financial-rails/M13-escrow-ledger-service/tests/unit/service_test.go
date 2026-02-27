package unit

import (
	"context"
	"testing"

	eventadapter "github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/adapters/events"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
)

func newService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{Holds: repos.Holds, Ledger: repos.Ledger, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup, Outbox: repos.Outbox, DomainEvents: eventadapter.NewMemoryDomainPublisher(), Analytics: eventadapter.NewMemoryAnalyticsPublisher(), DLQ: eventadapter.NewLoggingDLQPublisher()})
	return svc, repos
}

func TestCreateHoldAndReleaseEnqueueEvents(t *testing.T) {
	svc, repos := newService()
	actor := application.Actor{SubjectID: "svc_finance", Role: "system", RequestID: "req_1", IdempotencyKey: "idem-hold-1"}
	hold, err := svc.CreateHold(context.Background(), actor, application.CreateHoldInput{CampaignID: "camp_1", CreatorID: "user_1", Amount: 100})
	if err != nil { t.Fatalf("CreateHold: %v", err) }
	actor.IdempotencyKey = "idem-release-1"
	hold, err = svc.Release(context.Background(), actor, application.ReleaseInput{EscrowID: hold.EscrowID, Amount: 40})
	if err != nil { t.Fatalf("Release: %v", err) }
	if hold.Status != domain.HoldStatusPartialRelease { t.Fatalf("expected partial_release, got %s", hold.Status) }
	pending, err := repos.Outbox.ListPending(context.Background(), 10)
	if err != nil { t.Fatalf("ListPending: %v", err) }
	if len(pending) != 2 { t.Fatalf("expected 2 events, got %d", len(pending)) }
}

func TestRefundIdempotentReplay(t *testing.T) {
	svc, _ := newService()
	actor := application.Actor{SubjectID: "svc_finance", Role: "system", RequestID: "req_2", IdempotencyKey: "idem-hold-2"}
	hold, err := svc.CreateHold(context.Background(), actor, application.CreateHoldInput{CampaignID: "camp_2", CreatorID: "user_2", Amount: 50})
	if err != nil { t.Fatalf("CreateHold: %v", err) }
	actor.IdempotencyKey = "idem-refund-1"
	first, err := svc.Refund(context.Background(), actor, application.RefundInput{EscrowID: hold.EscrowID})
	if err != nil { t.Fatalf("Refund first: %v", err) }
	second, err := svc.Refund(context.Background(), actor, application.RefundInput{EscrowID: hold.EscrowID})
	if err != nil { t.Fatalf("Refund second: %v", err) }
	if first.EscrowID != second.EscrowID || first.RefundedAmount != second.RefundedAmount { t.Fatalf("idempotent replay mismatch: first=%+v second=%+v", first, second) }
}
