package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/domain"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	expectedClass := domain.CanonicalEventClass(envelope.EventType)
	if envelope.EventClass != "" && envelope.EventClass != expectedClass {
		return domain.ErrUnsupportedEventClass
	}
	expectedPath := domain.CanonicalPartitionKeyPath(envelope.EventType)
	if envelope.PartitionKeyPath != expectedPath {
		return domain.ErrInvalidEnvelope
	}
	if s.eventDedup != nil {
		dup, err := s.eventDedup.IsDuplicate(ctx, envelope.EventID, s.nowFn())
		if err != nil {
			return err
		}
		if dup {
			return nil
		}
		if err := s.eventDedup.MarkProcessed(ctx, envelope.EventID, envelope.EventType, s.nowFn().Add(s.cfg.EventDedupTTL)); err != nil {
			return err
		}
	}

	switch envelope.EventType {
	case domain.EventSocialAccountConnected:
		var payload contracts.SocialAccountConnectedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if strings.TrimSpace(payload.UserID) == "" || payload.UserID != envelope.PartitionKey {
			return domain.ErrInvalidEnvelope
		}
		at, err := parseRFC3339(payload.ConnectedAt)
		if err != nil {
			return domain.ErrInvalidEnvelope
		}
		row := domain.SocialAccount{
			SocialAccountID: "evt-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			UserID:          strings.TrimSpace(payload.UserID),
			Platform:        normalizeProvider(payload.Platform),
			Status:          domain.AccountStatusActive,
			ConnectedAt:     at,
			UpdatedAt:       s.nowFn(),
			Source:          "event_projection",
		}
		if row.Platform == "" {
			return domain.ErrInvalidEnvelope
		}
		if existing, err := s.accounts.GetByUserProvider(ctx, row.UserID, row.Platform); err == nil {
			row.SocialAccountID = existing.SocialAccountID
			row.Handle = existing.Handle
			row.ConnectedAt = existing.ConnectedAt
			return s.accounts.Update(ctx, row)
		}
		return s.accounts.Create(ctx, row)

	case domain.EventSocialStatusChanged:
		var payload contracts.SocialStatusChangedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if strings.TrimSpace(payload.UserID) == "" || payload.UserID != envelope.PartitionKey {
			return domain.ErrInvalidEnvelope
		}
		changedAt, err := parseRFC3339(payload.ChangedAt)
		if err != nil {
			return domain.ErrInvalidEnvelope
		}
		platform := normalizeProvider(payload.Platform)
		if platform == "" {
			return domain.ErrInvalidEnvelope
		}
		row := domain.SocialAccount{
			SocialAccountID: "evt-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			UserID:          strings.TrimSpace(payload.UserID),
			Platform:        platform,
			Status:          strings.TrimSpace(payload.Status),
			ConnectedAt:     changedAt,
			UpdatedAt:       s.nowFn(),
			Source:          "event_projection",
		}
		if row.Status == "" {
			row.Status = domain.AccountStatusActive
		}
		if existing, err := s.accounts.GetByUserProvider(ctx, row.UserID, row.Platform); err == nil {
			row.SocialAccountID = existing.SocialAccountID
			row.Handle = existing.Handle
			row.ConnectedAt = existing.ConnectedAt
			return s.accounts.Update(ctx, row)
		}
		return s.accounts.Create(ctx, row)

	case domain.EventSocialPostValidated:
		var payload contracts.SocialPostValidatedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if strings.TrimSpace(payload.UserID) == "" || payload.UserID != envelope.PartitionKey {
			return domain.ErrInvalidEnvelope
		}
		validatedAt, err := parseRFC3339(payload.ValidatedAt)
		if err != nil {
			return domain.ErrInvalidEnvelope
		}
		row := domain.PostValidation{
			ValidationID: "evt-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			UserID:       strings.TrimSpace(payload.UserID),
			Platform:     normalizeProvider(payload.Platform),
			PostID:       strings.TrimSpace(payload.PostID),
			IsValid:      true,
			ValidatedAt:  validatedAt,
			Source:       "event_projection",
		}
		if row.Platform == "" || row.PostID == "" {
			return domain.ErrInvalidEnvelope
		}
		return s.validations.UpsertByUserPlatformPost(ctx, row)

	case domain.EventSocialComplianceViolation:
		var payload contracts.SocialComplianceViolationPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if strings.TrimSpace(payload.UserID) == "" || payload.UserID != envelope.PartitionKey {
			return domain.ErrInvalidEnvelope
		}
		violationAt, err := parseRFC3339(payload.ViolationAt)
		if err != nil {
			return domain.ErrInvalidEnvelope
		}
		row := domain.PostValidation{
			ValidationID: "evt-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			UserID:       strings.TrimSpace(payload.UserID),
			Platform:     normalizeProvider(payload.Platform),
			PostID:       strings.TrimSpace(payload.PostID),
			IsValid:      false,
			Reason:       strings.TrimSpace(payload.Reason),
			ValidatedAt:  violationAt,
			Source:       "event_projection",
		}
		if row.Platform == "" || row.PostID == "" {
			return domain.ErrInvalidEnvelope
		}
		if row.Reason == "" {
			row.Reason = "policy_violation"
		}
		return s.validations.UpsertByUserPlatformPost(ctx, row)

	case domain.EventSocialFollowersSynced:
		var payload contracts.SocialFollowersSyncedPayload
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			return domain.ErrInvalidEnvelope
		}
		if strings.TrimSpace(payload.UserID) == "" || payload.UserID != envelope.PartitionKey {
			return domain.ErrInvalidEnvelope
		}
		syncedAt, err := parseRFC3339(payload.SyncedAt)
		if err != nil {
			return domain.ErrInvalidEnvelope
		}
		if payload.FollowerCount < 0 {
			return domain.ErrInvalidEnvelope
		}
		if s.metrics == nil {
			return nil
		}
		return s.metrics.Append(ctx, domain.SocialMetric{
			MetricID:      "evt-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
			UserID:        strings.TrimSpace(payload.UserID),
			Platform:      normalizeProvider(payload.Platform),
			FollowerCount: payload.FollowerCount,
			SyncedAt:      syncedAt,
		})
	default:
		return domain.ErrUnsupportedEventType
	}
}

func validateEnvelope(event contracts.EventEnvelope) error {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.EventType) == "" || event.OccurredAt.IsZero() {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.SourceService) == "" || strings.TrimSpace(event.TraceID) == "" || strings.TrimSpace(event.SchemaVersion) == "" {
		return domain.ErrInvalidEnvelope
	}
	if strings.TrimSpace(event.PartitionKeyPath) == "" || strings.TrimSpace(event.PartitionKey) == "" {
		return domain.ErrInvalidEnvelope
	}
	if len(event.Data) == 0 {
		return domain.ErrInvalidEnvelope
	}
	return nil
}

func parseRFC3339(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(raw))
}
