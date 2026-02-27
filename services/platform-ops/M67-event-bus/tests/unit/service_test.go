package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/application"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/contracts"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Topics:      repos.Topics,
		ACLs:        repos.ACLs,
		Offsets:     repos.Offsets,
		Schemas:     repos.Schemas,
		DLQ:         repos.DLQ,
		Metrics:     repos.Metrics,
		Idempotency: repos.Idempotency,
		EventDedup:  repos.EventDedup,
		Outbox:      repos.Outbox,
	})
}

func adminActor(key string) application.Actor {
	return application.Actor{SubjectID: "ops-admin", Role: "admin", IdempotencyKey: key}
}

func TestCreateTopicRejectsInvalidName(t *testing.T) {
	svc := newService()
	_, err := svc.CreateTopic(context.Background(), adminActor("idem-topic-invalid"), application.CreateTopicInput{
		TopicName: "new_submission",
	})
	if err != domain.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestPublishEventIdempotentWithRegisteredSchema(t *testing.T) {
	svc := newService()
	_, err := svc.RegisterSchema(context.Background(), adminActor("idem-schema-1"), application.RegisterSchemaInput{
		Subject: "submission.created-value",
		Schema:  `{"type":"record","name":"SubmissionCreated","fields":[{"name":"submission_id","type":"string"}]}`,
	})
	if err != nil {
		t.Fatalf("register schema: %v", err)
	}
	_, err = svc.CreateTopic(context.Background(), adminActor("idem-topic-1"), application.CreateTopicInput{
		TopicName: "submission.created",
	})
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	eventID := uuid.NewString()
	input := application.PublishInput{
		EventID:          eventID,
		EventType:        "submission.created",
		OccurredAt:       time.Now().UTC(),
		SourceService:    "submission-service",
		TraceID:          "trace-1",
		SchemaVersion:    "1.0",
		PartitionKeyPath: "data.submission_id",
		PartitionKey:     "sub-123",
		Format:           "avro",
		Data:             map[string]any{"submission_id": "sub-123"},
	}
	first, err := svc.PublishEvent(context.Background(), adminActor("idem-publish-1"), input)
	if err != nil {
		t.Fatalf("publish first: %v", err)
	}
	second, err := svc.PublishEvent(context.Background(), adminActor("idem-publish-1"), input)
	if err != nil {
		t.Fatalf("publish second: %v", err)
	}
	if first.EventID != second.EventID || first.Topic != second.Topic {
		t.Fatalf("expected idempotent replay same publish result")
	}
}

func TestCreateACLAndResetOffset(t *testing.T) {
	svc := newService()
	acl, err := svc.CreateACL(context.Background(), adminActor("idem-acl-1"), application.CreateACLInput{
		Principal:    "submission-service",
		ResourceType: "topic",
		ResourceName: "submission.created",
		Operations:   []string{"Write"},
	})
	if err != nil {
		t.Fatalf("create acl: %v", err)
	}
	if acl.ID == "" {
		t.Fatalf("expected acl id")
	}
	offset, err := svc.ResetConsumerOffset(context.Background(), adminActor("idem-off-1"), application.ResetOffsetInput{
		GroupID:   "analytics-group",
		Topic:     "submission.created",
		Partition: 0,
		Offset:    100,
		Reason:    "replay",
	})
	if err != nil {
		t.Fatalf("reset offset: %v", err)
	}
	if offset.Offset != 100 {
		t.Fatalf("expected offset 100, got %d", offset.Offset)
	}
}

func TestDLQReplayMarksMessagesReplayed(t *testing.T) {
	svc := newService()
	_, err := svc.AddDLQMessage(context.Background(), application.Actor{SubjectID: "system", Role: "system"}, domain.DLQMessage{
		SourceTopic:   "submission.created",
		ConsumerGroup: "analytics-group",
		ErrorType:     "processing_error",
		ErrorSummary:  "timeout",
		RetryCount:    3,
		EventID:       "evt-1",
		OriginalEvent: map[string]any{"event_id": "evt-1"},
	})
	if err != nil {
		t.Fatalf("add dlq: %v", err)
	}
	out, err := svc.ReplayDLQ(context.Background(), adminActor("idem-replay-1"), application.DLQReplayInput{
		SourceTopic: "submission.created",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("replay dlq: %v", err)
	}
	if out.Replayed != 1 {
		t.Fatalf("expected replayed 1, got %d", out.Replayed)
	}
	rows, err := svc.ListDLQ(context.Background(), application.Actor{SubjectID: "reader", Role: "admin"}, application.DLQListInput{
		SourceTopic:     "submission.created",
		IncludeReplayed: true,
	})
	if err != nil {
		t.Fatalf("list dlq: %v", err)
	}
	if len(rows) != 1 || rows[0].ReplayedAt == nil {
		t.Fatalf("expected dlq message marked replayed")
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
		Data:             []byte(`{"id":"id-1"}`),
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != domain.ErrUnsupportedEventType {
		t.Fatalf("expected unsupported event, got %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), env); err != nil {
		t.Fatalf("expected duplicate no-op, got %v", err)
	}
}
