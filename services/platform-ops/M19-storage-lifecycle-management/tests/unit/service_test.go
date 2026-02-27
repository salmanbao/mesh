package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Policies:    repos.Policies,
		Lifecycle:   repos.Lifecycle,
		Batches:     repos.Batches,
		Audits:      repos.Audits,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
}

func TestCreatePolicyIdempotentReplay(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "admin-1", Role: "admin", IdempotencyKey: "idem-pol-1"}
	in := application.CreatePolicyInput{
		Scope:           "approved_clips",
		TierFrom:        "STANDARD",
		TierTo:          "GLACIER_DEEP_ARCHIVE",
		AfterDays:       30,
		LegalHoldExempt: false,
	}
	first, err := svc.CreatePolicy(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("first create policy: %v", err)
	}
	second, err := svc.CreatePolicy(context.Background(), actor, in)
	if err != nil {
		t.Fatalf("second create policy: %v", err)
	}
	if first.PolicyID != second.PolicyID {
		t.Fatalf("expected same policy id on idempotent replay")
	}
}

func TestMoveToGlacierUpdatesAnalytics(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "svc-media", Role: "system", IdempotencyKey: "idem-move-1"}
	_, err := svc.MoveToGlacier(context.Background(), actor, application.MoveToGlacierInput{
		FileID:            "file-1",
		CampaignID:        "camp-1",
		SourceBucket:      "hot-bucket",
		SourceKey:         "raw/file-1.mp4",
		DestinationBucket: "cold-bucket",
		DestinationKey:    "archive/file-1.mp4",
		FileSizeBytes:     10 * 1024 * 1024 * 1024, // 10 GB
	})
	if err != nil {
		t.Fatalf("move to glacier: %v", err)
	}
	summary, err := svc.GetAnalyticsSummary(context.Background(), application.Actor{SubjectID: "ops-user", Role: "admin"})
	if err != nil {
		t.Fatalf("analytics summary: %v", err)
	}
	if summary.TotalObjects != 1 {
		t.Fatalf("expected 1 object, got %d", summary.TotalObjects)
	}
	if summary.ByTier[domain.TierGlacierDeepArchive] != 1 {
		t.Fatalf("expected 1 glacier object")
	}
}

func TestScheduleDeletionAndAuditQuery(t *testing.T) {
	svc := newService()
	actor := application.Actor{SubjectID: "svc-lifecycle", Role: "system", IdempotencyKey: "idem-del-1"}
	batch, err := svc.ScheduleDeletion(context.Background(), actor, application.ScheduleDeletionInput{
		CampaignID:       "camp-2",
		DeletionType:     domain.DeletionTypeRawFiles,
		DaysAfterClosure: 30,
		FileIDs:          []string{"file-a", "file-b", "file-a"},
	})
	if err != nil {
		t.Fatalf("schedule deletion: %v", err)
	}
	if batch.FileCount != 2 {
		t.Fatalf("expected deduped file count 2, got %d", batch.FileCount)
	}
	q, err := svc.QueryDeletionAudit(context.Background(), application.Actor{SubjectID: "admin-q", Role: "admin"}, application.AuditQueryInput{
		CampaignID: "camp-2",
		Action:     "soft_delete",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if len(q.Records) != 2 {
		t.Fatalf("expected 2 audit rows, got %d", len(q.Records))
	}
	if q.TotalFilesDeleted != 2 {
		t.Fatalf("expected total files deleted 2, got %d", q.TotalFilesDeleted)
	}
}

func TestHandleCanonicalEventUnsupportedDeduped(t *testing.T) {
	svc := newService()
	env := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "noncanonical.event",
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		SourceService:    "M00-Test",
		TraceID:          "trace-1",
		SchemaVersion:    "v1",
		PartitionKeyPath: "data.id",
		PartitionKey:     "id-1",
		Data:             []byte(`{\"id\":\"id-1\"}`),
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != domain.ErrUnsupportedEventType {
		t.Fatalf("expected unsupported event, got %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != nil {
		t.Fatalf("expected duplicate no-op, got %v", err)
	}
}
