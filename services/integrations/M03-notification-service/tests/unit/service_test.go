package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{Notifications: repos.Notifications, Preferences: repos.Preferences, Scheduled: repos.Scheduled, Idempotency: repos.Idempotency, EventDedup: repos.EventDedup})
}

func TestHandleCanonicalEventCreatesNotificationAndDedups(t *testing.T) {
	svc := newService()
	payload, _ := json.Marshal(map[string]any{"user_id": "u1", "registered_at": time.Now().UTC().Format(time.RFC3339)})
	e := contracts.EventEnvelope{EventID: "evt-1", EventType: domain.EventUserRegistered, EventClass: domain.CanonicalEventClassDomain, OccurredAt: time.Now().UTC(), PartitionKeyPath: "data.user_id", PartitionKey: "u1", SourceService: "M01-Auth-Service", TraceID: "trace-1", SchemaVersion: "v1", Data: payload}
	if err := svc.HandleCanonicalEvent(context.Background(), e); err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if err := svc.HandleCanonicalEvent(context.Background(), e); err != nil {
		t.Fatalf("dedup second handle: %v", err)
	}
	items, _, unread, err := svc.ListNotifications(context.Background(), application.Actor{SubjectID: "u1", Role: "user"}, application.ListNotificationsInput{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if unread != 1 || len(items) != 1 {
		t.Fatalf("expected 1 notification, got unread=%d len=%d", unread, len(items))
	}
}

func TestBulkActionIdempotency(t *testing.T) {
	svc := newService()
	for i := 0; i < 2; i++ {
		payload, _ := json.Marshal(map[string]any{"user_id": "u2", "registered_at": time.Now().UTC().Format(time.RFC3339)})
		e := contracts.EventEnvelope{EventID: "evt-x" + string(rune('a'+i)), EventType: domain.EventUserRegistered, EventClass: domain.CanonicalEventClassDomain, OccurredAt: time.Now().UTC(), PartitionKeyPath: "data.user_id", PartitionKey: "u2", SourceService: "M01", TraceID: "t", SchemaVersion: "v1", Data: payload}
		if err := svc.HandleCanonicalEvent(context.Background(), e); err != nil {
			t.Fatalf("seed event: %v", err)
		}
	}
	items, _, _, _ := svc.ListNotifications(context.Background(), application.Actor{SubjectID: "u2", Role: "user"}, application.ListNotificationsInput{})
	ids := []string{items[0].NotificationID, items[1].NotificationID}
	actor := application.Actor{SubjectID: "u2", Role: "user", IdempotencyKey: "idem-1"}
	updated1, failed1, err := svc.BulkAction(context.Background(), actor, application.BulkActionInput{Action: "mark_read", NotificationIDs: ids})
	if err != nil {
		t.Fatalf("bulk1: %v", err)
	}
	updated2, failed2, err := svc.BulkAction(context.Background(), actor, application.BulkActionInput{Action: "mark_read", NotificationIDs: ids})
	if err != nil {
		t.Fatalf("bulk2: %v", err)
	}
	if updated1 != 2 || updated2 != 2 || failed1 != 0 || failed2 != 0 {
		t.Fatalf("unexpected updated counts: %d/%d failed %d/%d", updated1, updated2, failed1, failed2)
	}
}

func TestHandlePhase6DependencyEvents(t *testing.T) {
	svc := newService()
	now := time.Now().UTC()
	cases := []struct {
		eventID   string
		eventType string
		payload   map[string]any
		keyPath   string
		key       string
	}{
		{
			eventID:   "evt-phase6-1",
			eventType: domain.EventPayoutFailed,
			payload:   map[string]any{"user_id": "u3", "payout_id": "p-1"},
			keyPath:   "data.payout_id",
			key:       "p-1",
		},
		{
			eventID:   "evt-phase6-2",
			eventType: domain.EventDisputeCreated,
			payload:   map[string]any{"dispute_id": "d-1", "user_id": "u3"},
			keyPath:   "data.dispute_id",
			key:       "d-1",
		},
	}
	for _, tc := range cases {
		data, _ := json.Marshal(tc.payload)
		err := svc.HandleCanonicalEvent(context.Background(), contracts.EventEnvelope{
			EventID:          tc.eventID,
			EventType:        tc.eventType,
			EventClass:       domain.CanonicalEventClassDomain,
			OccurredAt:       now,
			PartitionKeyPath: tc.keyPath,
			PartitionKey:     tc.key,
			SourceService:    "phase6-smoke",
			TraceID:          "trace-phase6",
			SchemaVersion:    "v1",
			Data:             data,
		})
		if err != nil {
			t.Fatalf("handle %s failed: %v", tc.eventType, err)
		}
	}
	items, _, unread, err := svc.ListNotifications(context.Background(), application.Actor{SubjectID: "u3", Role: "user"}, application.ListNotificationsInput{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(items) != 2 || unread != 2 {
		t.Fatalf("expected 2 unread notifications for phase6 smoke, got len=%d unread=%d", len(items), unread)
	}
}
