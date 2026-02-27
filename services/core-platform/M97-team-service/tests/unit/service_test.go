package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/adapters/events"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/contracts"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
)

func newTestService() (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Teams:        repos.Teams,
		Members:      repos.Members,
		Invites:      repos.Invites,
		Roles:        repos.Roles,
		AuditLogs:    repos.AuditLogs,
		Idempotency:  repos.Idempotency,
		EventDedup:   repos.EventDedup,
		Outbox:       repos.Outbox,
		DomainEvents: eventadapter.NewMemoryDomainPublisher(),
		Analytics:    eventadapter.NewMemoryAnalyticsPublisher(),
		DLQ:          eventadapter.NewLoggingDLQPublisher(),
	})
	return svc, repos
}

func TestTeamLifecycle_CreateInviteAcceptMembership(t *testing.T) {
	svc, repos := newTestService()
	ctx := context.Background()

	owner := application.Actor{SubjectID: "user_owner", Role: "user", IdempotencyKey: "idem-team-1", RequestID: "req-1"}
	team, err := svc.CreateTeam(ctx, owner, application.CreateTeamInput{ScopeType: "storefront", ScopeID: "store_1"})
	if err != nil {
		t.Fatalf("CreateTeam error: %v", err)
	}
	if team.OwnerID != "user_owner" {
		t.Fatalf("unexpected owner: %s", team.OwnerID)
	}

	owner.IdempotencyKey = "idem-invite-1"
	invite, err := svc.CreateInvite(ctx, owner, application.CreateInviteInput{TeamID: team.TeamID, Email: "member@example.com", Role: "editor"})
	if err != nil {
		t.Fatalf("CreateInvite error: %v", err)
	}
	if invite.Status != domain.InviteStatusPending {
		t.Fatalf("expected pending invite, got %s", invite.Status)
	}

	memberActor := application.Actor{SubjectID: "user_member", Role: "user", IdempotencyKey: "idem-accept-1", RequestID: "req-2"}
	accepted, err := svc.AcceptInvite(ctx, memberActor, invite.InviteID)
	if err != nil {
		t.Fatalf("AcceptInvite error: %v", err)
	}
	if accepted.Status != "accepted" || accepted.MemberRole != "editor" {
		t.Fatalf("unexpected accept result: %+v", accepted)
	}

	check, err := svc.CheckMembership(ctx, application.Actor{SubjectID: "svc", Role: "system", RequestID: "req-3"}, application.MembershipCheckInput{TeamID: team.TeamID, UserID: "user_member", Permission: "member.view"})
	if err != nil {
		t.Fatalf("CheckMembership error: %v", err)
	}
	if !check.Allowed || check.Role != "editor" {
		t.Fatalf("unexpected membership check: %+v", check)
	}

	pending, err := repos.Outbox.ListPending(ctx, 20)
	if err != nil {
		t.Fatalf("ListPending error: %v", err)
	}
	if len(pending) < 5 {
		t.Fatalf("expected outbox events for team lifecycle, got %d", len(pending))
	}
	if err := svc.FlushOutbox(ctx); err != nil {
		t.Fatalf("FlushOutbox error: %v", err)
	}
	pending, err = repos.Outbox.ListPending(ctx, 20)
	if err != nil {
		t.Fatalf("ListPending after flush error: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected empty pending outbox, got %d", len(pending))
	}
}

func TestCreateTeam_IdempotencyConflict(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	actor := application.Actor{SubjectID: "user_owner", Role: "user", IdempotencyKey: "idem-same", RequestID: "req-1"}
	if _, err := svc.CreateTeam(ctx, actor, application.CreateTeamInput{ScopeType: "account", ScopeID: "acct_1"}); err != nil {
		t.Fatalf("first CreateTeam error: %v", err)
	}
	_, err := svc.CreateTeam(ctx, actor, application.CreateTeamInput{ScopeType: "account", ScopeID: "acct_2"})
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestHandleCanonicalEvent_UnsupportedButDeduped(t *testing.T) {
	svc, repos := newTestService()
	payload, _ := json.Marshal(map[string]any{"team_id": "team_1", "foo": "bar"})
	env := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "unknown.event",
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.team_id",
		PartitionKey:     "team_1",
		SourceService:    "M00-Test",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		Data:             payload,
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != domain.ErrUnsupportedEventType {
		t.Fatalf("expected unsupported event, got %v", err)
	}
	dup, err := repos.EventDedup.IsDuplicate(context.Background(), "evt-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("IsDuplicate error: %v", err)
	}
	if !dup {
		t.Fatalf("expected event dedup mark")
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != nil {
		t.Fatalf("expected duplicate to be ignored, got %v", err)
	}
}
