package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/ports"
)

type fakeOwnerAPI struct {
	rows []ports.VerificationAccount
}

func (f *fakeOwnerAPI) ListUserAccounts(context.Context, string) ([]ports.VerificationAccount, error) {
	return append([]ports.VerificationAccount{}, f.rows...), nil
}

func newServiceForTest(owner ports.SocialVerificationOwnerAPI) (*application.Service, *postgres.Repositories) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:    "M30-Social-Integration-Service",
			Version:        "test",
			IdempotencyTTL: 7 * 24 * time.Hour,
			EventDedupTTL:  7 * 24 * time.Hour,
		},
		Accounts:    repos.Accounts,
		Validations: repos.Validations,
		Metrics:     repos.Metrics,
		OwnerAPI:    owner,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
	return svc, repos
}

func TestConnectAccountIdempotency(t *testing.T) {
	svc, _ := newServiceForTest(&fakeOwnerAPI{})
	ctx := context.Background()
	actor := application.Actor{SubjectID: "user-1", Role: "developer", IdempotencyKey: "idem-1", RequestID: "req-1"}
	in := application.ConnectAccountInput{Platform: "instagram"}

	first, err := svc.ConnectAccount(ctx, actor, in)
	if err != nil {
		t.Fatalf("connect account: %v", err)
	}
	second, err := svc.ConnectAccount(ctx, actor, in)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if first.SocialAccountID != second.SocialAccountID {
		t.Fatalf("expected same account id, got %s and %s", first.SocialAccountID, second.SocialAccountID)
	}

	_, err = svc.ConnectAccount(ctx, actor, application.ConnectAccountInput{Platform: "youtube", Handle: "other"})
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestListAccountsMergesOwnerAPI(t *testing.T) {
	owner := &fakeOwnerAPI{rows: []ports.VerificationAccount{{
		UserID:      "user-1",
		Platform:    "youtube",
		Handle:      "yt-user",
		Status:      "active",
		ConnectedAt: time.Now().Add(-time.Hour).UTC(),
	}}}
	svc, _ := newServiceForTest(owner)
	ctx := context.Background()
	actor := application.Actor{SubjectID: "user-1", Role: "developer", IdempotencyKey: "idem-2", RequestID: "req-2"}
	_, err := svc.ConnectAccount(ctx, actor, application.ConnectAccountInput{Platform: "instagram", Handle: "ig-user"})
	if err != nil {
		t.Fatalf("connect local account: %v", err)
	}

	rows, err := svc.ListAccounts(ctx, actor, "")
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 accounts merged, got %d", len(rows))
	}
}

func TestHandleCanonicalEventEnvelopeAndDedup(t *testing.T) {
	svc, repos := newServiceForTest(&fakeOwnerAPI{})
	ctx := context.Background()

	data, _ := json.Marshal(contracts.SocialFollowersSyncedPayload{
		UserID:        "user-99",
		Platform:      "instagram",
		FollowerCount: 120,
		SyncedAt:      time.Now().UTC().Format(time.RFC3339),
	})
	e := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        domain.EventSocialFollowersSynced,
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		PartitionKeyPath: "data.user_id",
		PartitionKey:     "user-99",
		SourceService:    "M10-Social-Integration-Verification-Service",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		Data:             data,
	}
	if err := svc.HandleCanonicalEvent(ctx, e); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	dup, err := repos.EventDedup.IsDuplicate(ctx, "evt-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("dedup lookup: %v", err)
	}
	if !dup {
		t.Fatalf("expected event to be marked as duplicate")
	}
	if err := svc.HandleCanonicalEvent(ctx, e); err != nil {
		t.Fatalf("duplicate event should be ignored, got %v", err)
	}

	bad := e
	bad.EventID = "evt-2"
	bad.PartitionKey = "another-user"
	if err := svc.HandleCanonicalEvent(ctx, bad); err != domain.ErrInvalidEnvelope {
		t.Fatalf("expected invalid envelope, got %v", err)
	}
}
