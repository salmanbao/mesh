package unit

import (
	"context"
	"testing"
	"time"

	eventsadapter "github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
)

func newService() (*application.Service, *postgres.Repositories, *eventsadapter.MemoryDomainPublisher, *eventsadapter.MemoryAnalyticsPublisher) {
	repos := postgres.NewRepositories()
	domainPub := eventsadapter.NewMemoryDomainPublisher()
	analyticsPub := eventsadapter.NewMemoryAnalyticsPublisher()
	svc := application.NewService(application.Dependencies{
		Config:   application.Config{ServiceName: "M44-Resolution-Center", IdempotencyTTL: 7 * 24 * time.Hour, EventDedupTTL: 7 * 24 * time.Hour, OutboxFlushBatchSize: 100, DefaultCurrency: "USD"},
		Disputes: repos.Disputes, Messages: repos.Messages, Evidence: repos.Evidence, Approvals: repos.Approvals, AuditLogs: repos.AuditLogs, StateHistory: repos.StateHistory, Mediation: repos.Mediation, Rules: repos.Rules, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup, Outbox: repos.Outbox,
		Moderation: grpcadapter.NewModerationClient(""), DomainEvents: domainPub, Analytics: analyticsPub, DLQ: eventsadapter.NewLoggingDLQPublisher(),
	})
	return svc, repos, domainPub, analyticsPub
}

func TestCreateDisputeIdempotency(t *testing.T) {
	t.Parallel()
	svc, _, _, _ := newService()
	actor := application.Actor{SubjectID: "user-1", Role: "user", RequestID: "req-1", IdempotencyKey: "idem-dispute-1"}
	input := application.CreateDisputeInput{DisputeType: domain.DisputeTypeRefundRequest, TransactionID: "txn-1", ReasonCategory: "service_not_received", JustificationText: "This service was not delivered and support did not respond within the promised timeframe.", RequestedAmount: 150}
	first, err := svc.CreateDispute(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first create dispute: %v", err)
	}
	second, err := svc.CreateDispute(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second create dispute: %v", err)
	}
	if first.DisputeID != second.DisputeID {
		t.Fatalf("expected idempotent replay same dispute id")
	}
}

func TestApproveDisputeEmitsRefundAndAnalyticsResolution(t *testing.T) {
	t.Parallel()
	svc, repos, domainPub, analyticsPub := newService()
	user := application.Actor{SubjectID: "user-1", Role: "user", RequestID: "req-u", IdempotencyKey: "idem-dispute-2"}
	dispute, err := svc.CreateDispute(context.Background(), user, application.CreateDisputeInput{DisputeType: domain.DisputeTypeRefundRequest, TransactionID: "txn-2", ReasonCategory: "duplicate_charge", JustificationText: "I was charged twice for the same transaction and have included screenshots to confirm the duplicate billing issue.", RequestedAmount: 25})
	if err != nil {
		t.Fatalf("create dispute: %v", err)
	}
	staff := application.Actor{SubjectID: "agent-1", Role: "agent", RequestID: "req-a", IdempotencyKey: "idem-approve-1"}
	resolved, err := svc.ApproveDispute(context.Background(), staff, dispute.DisputeID, application.ApproveDisputeInput{RefundAmount: 25, ApprovalReason: "duplicate confirmed", ResolutionNotes: "full refund"})
	if err != nil {
		t.Fatalf("approve dispute: %v", err)
	}
	if resolved.Status != domain.DisputeStatusResolved {
		t.Fatalf("expected resolved status")
	}
	pending, err := repos.Outbox.ListPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("outbox list pending: %v", err)
	}
	if len(pending) < 2 {
		t.Fatalf("expected dispute.created and transaction.refunded in outbox")
	}
	if err := svc.FlushOutbox(context.Background()); err != nil {
		t.Fatalf("flush outbox: %v", err)
	}
	if len(domainPub.Events()) == 0 {
		t.Fatalf("expected domain publisher events after flush")
	}
	if len(analyticsPub.Events()) == 0 {
		t.Fatalf("expected analytics dispute.resolved event")
	}
}

func TestHandleCanonicalEventDedup(t *testing.T) {
	t.Parallel()
	svc, _, _, _ := newService()
	event := contracts.EventEnvelope{EventID: "evt-1", EventType: domain.EventSubmissionApproved, EventClass: domain.CanonicalEventClassDomain, OccurredAt: time.Now().UTC(), PartitionKeyPath: "data.submission_id", PartitionKey: "sub-1", SourceService: "M26-Submission-Service", TraceID: "trace-1", SchemaVersion: "v1", Data: []byte(`{"submission_id":"sub-1","user_id":"user-1","campaign_id":"cmp-1","approved_at":"2026-02-26T00:00:00Z"}`)}
	if err := svc.HandleCanonicalEvent(context.Background(), event); err != nil {
		t.Fatalf("first handle event: %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), event); err != nil {
		t.Fatalf("duplicate handle should be no-op: %v", err)
	}
}
