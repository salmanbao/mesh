package unit

import (
	"context"
	"io"
	"testing"
	"time"

	eventadapter "github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/adapters/events"
	grpcadapter "github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
)

type testDeps struct {
	service *application.Service
	repos   *postgres.Repositories
	queue   *eventadapter.MemoryJobQueue
	dlq     *eventadapter.MemoryDLQPublisher
}

func newService() testDeps {
	repos := postgres.NewRepositories()
	queue := eventadapter.NewMemoryJobQueue()
	dlq := eventadapter.NewMemoryDLQPublisher()
	service := application.NewService(application.Dependencies{
		Config: application.Config{ServiceName: "M06-Media-Processing-Pipeline", IdempotencyTTL: 7 * 24 * time.Hour, EventDedupTTL: 7 * 24 * time.Hour},
		Assets: repos.Assets, Jobs: repos.Jobs, Outputs: repos.Outputs, Thumbnails: repos.Thumbnails, Watermarks: repos.Watermarks, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup,
		Campaign: grpcadapter.NewCampaignClient(""),
		Queue:    queue,
		DLQ:      dlq,
	})
	return testDeps{service: service, repos: repos, queue: queue, dlq: dlq}
}

func TestCreateUploadIdempotent(t *testing.T) {
	deps := newService()
	svc := deps.service
	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "media-upload:sub-1:abc"}
	input := application.CreateUploadInput{SubmissionID: "sub-1", FileName: "clip.mp4", MIMEType: "video/mp4", FileSize: 1000, ChecksumSHA256: "abc"}

	first, err := svc.CreateUpload(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first create upload: %v", err)
	}
	second, err := svc.CreateUpload(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second create upload: %v", err)
	}
	if first.AssetID != second.AssetID {
		t.Fatalf("expected same asset for idempotent replay")
	}
}

func TestCreateUploadRequiresCanonicalIdempotencyKey(t *testing.T) {
	deps := newService()
	svc := deps.service
	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "bad-key"}
	input := application.CreateUploadInput{SubmissionID: "sub-1", FileName: "clip.mp4", MIMEType: "video/mp4", FileSize: 1000, ChecksumSHA256: "abc"}
	_, err := svc.CreateUpload(context.Background(), actor, input)
	if err != domain.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestRetryRequiresAdmin(t *testing.T) {
	deps := newService()
	svc := deps.service
	_, err := svc.RetryAsset(context.Background(), application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "retry"}, application.RetryAssetInput{AssetID: "asset-1"})
	if err != domain.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestProcessQueueGeneratesVariantsAndCompletes(t *testing.T) {
	deps := newService()
	svc := deps.service
	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "media-upload:sub-2:def"}
	asset, err := svc.CreateUpload(context.Background(), actor, application.CreateUploadInput{SubmissionID: "sub-2", FileName: "clip.mp4", MIMEType: "video/mp4", FileSize: 1000, ChecksumSHA256: "def"})
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}

	for i := 0; i < 32; i++ {
		err = svc.ProcessNextJob(context.Background())
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("process job: %v", err)
		}
	}
	status, err := svc.GetAssetStatus(context.Background(), application.Actor{SubjectID: "user-1"}, asset.AssetID)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status.Status != string(domain.AssetStatusCompleted) {
		t.Fatalf("expected completed status, got %s", status.Status)
	}
	if len(status.Outputs) != 6 {
		t.Fatalf("expected 6 outputs, got %d", len(status.Outputs))
	}
	if len(status.Thumbnails) != 9 {
		t.Fatalf("expected 9 thumbnails, got %d", len(status.Thumbnails))
	}
}

func TestRetryCompletedAssetNoOp(t *testing.T) {
	deps := newService()
	svc := deps.service
	uploadActor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "media-upload:sub-3:ghi"}
	asset, err := svc.CreateUpload(context.Background(), uploadActor, application.CreateUploadInput{
		SubmissionID: "sub-3", FileName: "clip.mp4", MIMEType: "video/mp4", FileSize: 1000, ChecksumSHA256: "ghi",
	})
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	for i := 0; i < 32; i++ {
		err = svc.ProcessNextJob(context.Background())
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("process job: %v", err)
		}
	}
	retry, err := svc.RetryAsset(context.Background(), application.Actor{
		SubjectID:      "admin-1",
		Role:           "admin",
		IdempotencyKey: "media-retry:" + asset.AssetID + ":" + time.Now().UTC().Format("200601021504"),
	}, application.RetryAssetInput{AssetID: asset.AssetID})
	if err != nil {
		t.Fatalf("retry completed asset: %v", err)
	}
	if retry.JobsRestarted != 0 {
		t.Fatalf("expected no-op retry with 0 jobs, got %d", retry.JobsRestarted)
	}
}

func TestRetryableJobFailurePublishesDLQAfterMaxAttempts(t *testing.T) {
	deps := newService()
	now := time.Now().UTC()
	asset := domain.MediaAsset{
		AssetID:          "asset-retry",
		SubmissionID:     "sub-retry",
		OriginalFilename: "clip.mp4",
		MIMEType:         "video/mp4",
		FileSize:         1000,
		SourceS3URL:      "s3://media-raw/asset-retry",
		Status:           domain.AssetStatusProcessing,
		ApprovalStatus:   domain.ApprovalStatusPending,
		ChecksumSHA256:   "retry-hash",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := deps.repos.Assets.Create(context.Background(), asset); err != nil {
		t.Fatalf("seed asset: %v", err)
	}
	job := domain.MediaJob{
		JobID:    "job-retry",
		AssetID:  asset.AssetID,
		JobType:  domain.JobType("custom_transient"),
		Status:   domain.JobStatusQueued,
		QueuedAt: now,
	}
	if err := deps.repos.Jobs.CreateMany(context.Background(), []domain.MediaJob{job}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if err := deps.queue.Enqueue(context.Background(), job.JobID); err != nil {
		t.Fatalf("enqueue job: %v", err)
	}
	for i := 0; i < 6; i++ {
		err := deps.service.ProcessNextJob(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("process job: %v", err)
		}
	}
	updated, err := deps.repos.Jobs.GetByID(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("load job: %v", err)
	}
	if updated.Status != domain.JobStatusFailed {
		t.Fatalf("expected failed job, got %s", updated.Status)
	}
	if updated.Attempts != domain.MaxJobAttempts {
		t.Fatalf("expected %d attempts, got %d", domain.MaxJobAttempts, updated.Attempts)
	}
	records := deps.dlq.Records()
	if len(records) != 1 {
		t.Fatalf("expected 1 DLQ record, got %d", len(records))
	}
	if records[0].JobID != job.JobID {
		t.Fatalf("unexpected dlq job id: %s", records[0].JobID)
	}
}

func TestHandleInternalEventDeduplicatesDomainClass(t *testing.T) {
	deps := newService()
	event := contracts.EventEnvelope{
		EventID:          "evt-1",
		EventType:        "media.asset.reconcile.requested",
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		SourceService:    "test",
		TraceID:          "trace-1",
		SchemaVersion:    "1.0",
		PartitionKeyPath: "data.asset_id",
		PartitionKey:     "a1",
		Data:             map[string]any{"asset_id": "a1"},
	}
	if err := deps.service.HandleInternalEvent(context.Background(), event); err != nil {
		t.Fatalf("first handle event: %v", err)
	}
	if err := deps.service.HandleInternalEvent(context.Background(), event); err != nil {
		t.Fatalf("duplicate handle event: %v", err)
	}
	event.EventType = "unknown.event"
	if err := deps.service.HandleInternalEvent(context.Background(), event); err != domain.ErrUnsupportedEvent {
		t.Fatalf("expected unsupported event, got %v", err)
	}
}

func TestHandleInternalEventRejectsInvalidPartitionInvariant(t *testing.T) {
	deps := newService()
	event := contracts.EventEnvelope{
		EventID:          "evt-2",
		EventType:        "media.asset.reconcile.requested",
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		SourceService:    "test",
		TraceID:          "trace-2",
		SchemaVersion:    "1.0",
		PartitionKeyPath: "data.asset_id",
		PartitionKey:     "wrong",
		Data:             map[string]any{"asset_id": "a2"},
	}
	if err := deps.service.HandleInternalEvent(context.Background(), event); err != domain.ErrUnsupportedEvent {
		t.Fatalf("expected unsupported event for partition invariant mismatch, got %v", err)
	}
}

func TestHandleInternalEventRejectsMissingRequiredEnvelopeFields(t *testing.T) {
	deps := newService()
	event := contracts.EventEnvelope{
		EventID:          "evt-3",
		EventType:        "media.asset.reconcile.requested",
		EventClass:       domain.CanonicalEventClassDomain,
		OccurredAt:       time.Now().UTC(),
		SourceService:    "test",
		SchemaVersion:    "1.0",
		PartitionKeyPath: "data.asset_id",
		PartitionKey:     "a3",
		Data:             map[string]any{"asset_id": "a3"},
	}
	if err := deps.service.HandleInternalEvent(context.Background(), event); err != domain.ErrUnsupportedEvent {
		t.Fatalf("expected unsupported event for missing envelope fields, got %v", err)
	}
}
