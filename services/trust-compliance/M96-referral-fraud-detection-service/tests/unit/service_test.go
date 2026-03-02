package unit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/application"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/ports"
)

func newTestService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		ReferralEvents: repos.ReferralEvents,
		Decisions:      repos.Decisions,
		Policies:       repos.Policies,
		Fingerprints:   repos.Fingerprints,
		Clusters:       repos.Clusters,
		Disputes:       repos.Disputes,
		AuditLogs:      repos.AuditLogs,
		Idempotency:    repos.Idempotency,
		EventDedup:     repos.EventDedup,
		Outbox:         repos.Outbox,
		Affiliate:      grpcadapter.NewAffiliateClient("stub"),
		DomainEvents:   eventadapter.NewMemoryDomainPublisher(),
		Analytics:      eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:            eventadapter.NewLoggingDLQPublisher(),
	})
	return svc, repos
}

func TestHandleCanonicalEvent_ClickTracked(t *testing.T) {
	svc, repos := newTestService()
	payload, _ := json.Marshal(map[string]any{"affiliate_id": "aff_1", "link_id": "lnk_1", "referrer_url": "https://example.com", "ip_hash": "iphash-12345678", "tracked_at": "2026-02-01T00:00:00Z"})
	err := svc.HandleCanonicalEvent(context.Background(), contracts.EventEnvelope{EventID: "evt_1", EventType: domain.EventAffiliateClickTracked, EventClass: domain.CanonicalEventClassDomain, OccurredAt: time.Now().UTC(), PartitionKeyPath: "data.affiliate_id", PartitionKey: "aff_1", SourceService: "M89-Affiliate-Service", TraceID: "trace_1", SchemaVersion: "v1", Data: payload})
	if err != nil {
		t.Fatalf("HandleCanonicalEvent error: %v", err)
	}
	if repos.ReferralEvents.Count() != 1 {
		t.Fatalf("expected 1 referral event, got %d", repos.ReferralEvents.Count())
	}
	if _, err := repos.Decisions.GetByEventID(context.Background(), "evt_1"); err != nil {
		t.Fatalf("expected decision created: %v", err)
	}
}

func TestHandleCanonicalEvent_Dedup(t *testing.T) {
	svc, repos := newTestService()
	payload, _ := json.Marshal(map[string]any{"user_id": "user_1", "registered_at": "2026-02-01T00:00:00Z"})
	env := contracts.EventEnvelope{EventID: "evt_dup", EventType: domain.EventUserRegistered, EventClass: domain.CanonicalEventClassDomain, OccurredAt: time.Now().UTC(), PartitionKeyPath: "data.user_id", PartitionKey: "user_1", SourceService: "M01-Authentication-Service", TraceID: "trace_1", SchemaVersion: "v1", Data: payload}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != nil {
		t.Fatalf("first event err: %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != nil {
		t.Fatalf("second event err: %v", err)
	}
	if repos.ReferralEvents.Count() != 1 {
		t.Fatalf("expected dedup to keep 1 event, got %d", repos.ReferralEvents.Count())
	}
}

func TestScoreReferralAndDisputeIdempotency(t *testing.T) {
	svc, _ := newTestService()
	actor := application.Actor{SubjectID: "admin_1", Role: "admin", IdempotencyKey: "idem-score-1", RequestID: "req_1"}
	dec, err := svc.ScoreReferral(context.Background(), actor, application.ScoreInput{EventID: "evt_api_1", EventType: domain.EventAffiliateClickTracked, AffiliateID: "aff_1", ClickIP: "1.2.3.4", UserAgent: "Mozilla", OccurredAt: time.Now().UTC().Format(time.RFC3339)})
	if err != nil {
		t.Fatalf("ScoreReferral err: %v", err)
	}
	_, err = svc.ScoreReferral(context.Background(), actor, application.ScoreInput{EventID: "evt_api_2", EventType: domain.EventAffiliateClickTracked, AffiliateID: "aff_1", ClickIP: "1.2.3.4", UserAgent: "Mozilla", OccurredAt: time.Now().UTC().Format(time.RFC3339)})
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}

	disputeActor := application.Actor{SubjectID: "admin_1", Role: "admin", IdempotencyKey: "idem-dispute-1", RequestID: "req_2"}
	dispute, err := svc.SubmitDispute(context.Background(), disputeActor, application.SubmitDisputeInput{DecisionID: dec.DecisionID, SubmittedBy: "user_1", EvidenceURL: "https://example.com/evidence"})
	if err != nil {
		t.Fatalf("SubmitDispute err: %v", err)
	}
	if dispute.Status != "submitted" {
		t.Fatalf("expected submitted dispute")
	}
	_, err = svc.SubmitDispute(context.Background(), application.Actor{SubjectID: "admin_1", Role: "admin", IdempotencyKey: "idem-dispute-2"}, application.SubmitDisputeInput{DecisionID: dec.DecisionID, SubmittedBy: "user_1", EvidenceURL: "https://example.com/evidence"})
	if err != domain.ErrConflict {
		t.Fatalf("expected duplicate dispute conflict, got %v", err)
	}
}

func TestFlushOutboxReturnsAnalyticsPublishError(t *testing.T) {
	t.Parallel()

	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		ReferralEvents: repos.ReferralEvents,
		Decisions:      repos.Decisions,
		Policies:       repos.Policies,
		Fingerprints:   repos.Fingerprints,
		Clusters:       repos.Clusters,
		Disputes:       repos.Disputes,
		AuditLogs:      repos.AuditLogs,
		Idempotency:    repos.Idempotency,
		EventDedup:     repos.EventDedup,
		Outbox:         repos.Outbox,
		Affiliate:      grpcadapter.NewAffiliateClient("stub"),
		DomainEvents:   eventadapter.NewMemoryDomainPublisher(),
		Analytics:      failingAnalyticsPublisher{err: errors.New("publish failed")},
		DLQ:            eventadapter.NewLoggingDLQPublisher(),
	})

	now := time.Now().UTC()
	err := repos.Outbox.Enqueue(context.Background(), ports.OutboxRecord{
		RecordID:   "outbox-analytics-1",
		EventClass: domain.CanonicalEventClassAnalyticsOnly,
		Envelope: contracts.EventEnvelope{
			EventID:          "evt-analytics-1",
			EventType:        "referral.fraud.analytics.internal",
			EventClass:       domain.CanonicalEventClassAnalyticsOnly,
			OccurredAt:       now,
			PartitionKeyPath: "data.event_id",
			PartitionKey:     "evt-analytics-1",
			SourceService:    "M96-Referral-Fraud-Detection-Service",
			TraceID:          "trace-analytics-1",
			SchemaVersion:    "1.0",
			Data:             []byte(`{"event_id":"evt-analytics-1"}`),
		},
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("enqueue outbox record: %v", err)
	}

	if err := svc.FlushOutbox(context.Background()); err == nil {
		t.Fatalf("expected analytics publish error from flush outbox")
	}

	pending, err := repos.Outbox.ListPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("list pending outbox: %v", err)
	}
	if len(pending) != 1 || pending[0].RecordID != "outbox-analytics-1" {
		t.Fatalf("expected analytics record to remain pending after publish failure, got %#v", pending)
	}
}

type failingAnalyticsPublisher struct {
	err error
}

func (p failingAnalyticsPublisher) PublishAnalytics(context.Context, contracts.EventEnvelope) error {
	return p.err
}
