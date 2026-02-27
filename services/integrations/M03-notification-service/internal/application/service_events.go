package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/domain"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	expectedClass := domain.CanonicalEventClass(envelope.EventType)
	if strings.TrimSpace(envelope.EventClass) != "" && envelope.EventClass != expectedClass {
		return domain.ErrUnsupportedEventClass
	}
	expectedPath := domain.CanonicalPartitionKeyPath(envelope.EventType)
	if err := validatePartitionKeyInvariant(envelope, expectedPath); err != nil {
		return err
	}
	if envelope.EventType == domain.EventAuth2FARequired {
		if s.nowFn().Sub(envelope.OccurredAt) > 10*time.Minute {
			return nil
		}
	}
	now := s.nowFn()
	dup, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, now)
	if err != nil {
		return err
	}
	if dup {
		return nil
	}
	if err := s.ingestNotificationEvent(ctx, envelope); err != nil {
		return err
	}
	return s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, now.Add(s.cfg.EventDedupTTL))
}

func (s *Service) ingestNotificationEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	var payload map[string]any
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return domain.ErrInvalidEnvelope
	}
	userID := firstNonEmpty(
		asString(payload["user_id"]),
		asString(payload["creator_id"]),
	)
	if userID == "" {
		userID = "system"
	}
	n := domain.Notification{
		NotificationID:  newNotificationID(),
		UserID:          userID,
		Type:            envelope.EventType,
		Title:           humanizeEventType(envelope.EventType),
		Body:            fmt.Sprintf("Notification generated from %s", envelope.EventType),
		Metadata:        summarizePayload(payload),
		SourceEventID:   envelope.EventID,
		SourceEventType: envelope.EventType,
		CreatedAt:       s.nowFn(),
	}
	return s.notifications.Create(ctx, n)
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func validatePartitionKeyInvariant(event contracts.EventEnvelope, expectedPath string) error {
	if strings.TrimSpace(expectedPath) == "" || event.PartitionKeyPath != expectedPath {
		return domain.ErrInvalidEnvelope
	}
	field := strings.TrimPrefix(expectedPath, "data.")
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return domain.ErrInvalidEnvelope
	}
	v, ok := payload[field]
	if !ok || fmt.Sprint(v) != event.PartitionKey {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func summarizePayload(in map[string]any) map[string]string {
	keys := []string{"user_id", "campaign_id", "submission_id", "payout_id", "transaction_id", "dispute_id", "reason"}
	out := map[string]string{}
	for _, k := range keys {
		if v := asString(in[k]); v != "" {
			out[k] = v
		}
	}
	return out
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func humanizeEventType(t string) string {
	t = strings.ReplaceAll(t, ".", " ")
	t = strings.ReplaceAll(t, "_", " ")
	return strings.Title(t)
}
