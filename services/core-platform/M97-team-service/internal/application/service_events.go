package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/contracts"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/ports"
)

func (s *Service) HandleCanonicalEvent(ctx context.Context, envelope contracts.EventEnvelope) error {
	if err := validateEnvelope(envelope); err != nil {
		return err
	}
	if err := validatePartitionKeyInvariant(envelope); err != nil {
		return err
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
	if !domain.IsCanonicalInputEvent(envelope.EventType) {
		return domain.ErrUnsupportedEventType
	}
	return domain.ErrUnsupportedEventType
}

func (s *Service) FlushOutbox(ctx context.Context) error {
	if s.outbox == nil {
		return nil
	}
	pending, err := s.outbox.ListPending(ctx, s.cfg.OutboxFlushBatchSize)
	if err != nil {
		return err
	}
	for _, rec := range pending {
		now := s.nowFn()
		switch rec.EventClass {
		case domain.CanonicalEventClassDomain:
			if s.domainEvents != nil {
				if err := s.domainEvents.PublishDomain(ctx, rec.Envelope); err != nil {
					if s.dlq != nil {
						n := s.nowFn()
						_ = s.dlq.PublishDLQ(ctx, contracts.DLQRecord{OriginalEvent: rec.Envelope, ErrorSummary: err.Error(), RetryCount: 1, FirstSeenAt: n, LastErrorAt: n, SourceTopic: rec.Envelope.EventType, DLQTopic: "team-service.dlq", TraceID: rec.Envelope.TraceID})
					}
					return err
				}
			}
		case domain.CanonicalEventClassAnalyticsOnly:
			if s.analytics != nil {
				_ = s.analytics.PublishAnalytics(ctx, rec.Envelope)
			}
		default:
			return fmt.Errorf("%w: %s", domain.ErrUnsupportedEventClass, rec.EventClass)
		}
		if err := s.outbox.MarkSent(ctx, rec.RecordID, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) enqueueEvent(ctx context.Context, eventType, traceID string, data any, teamID string, now time.Time) error {
	if s.outbox == nil {
		return nil
	}
	if !domain.IsCanonicalEmittedEvent(eventType) {
		return domain.ErrUnsupportedEventType
	}
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return domain.ErrInvalidInput
	}
	b, err := json.Marshal(data)
	if err != nil {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = uuid.NewString()
	}
	env := contracts.EventEnvelope{
		EventID:          uuid.NewString(),
		EventType:        eventType,
		EventClass:       domain.CanonicalEventClass(eventType),
		OccurredAt:       now,
		PartitionKeyPath: domain.CanonicalPartitionKeyPath(eventType),
		PartitionKey:     teamID,
		SourceService:    s.cfg.ServiceName,
		TraceID:          traceID,
		SchemaVersion:    "v1",
		Data:             b,
	}
	return s.outbox.Enqueue(ctx, ports.OutboxRecord{RecordID: uuid.NewString(), EventClass: env.EventClass, Envelope: env, CreatedAt: now})
}

func (s *Service) enqueueTeamCreated(ctx context.Context, team domain.Team, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventTeamCreated, traceID, contracts.TeamCreatedPayload{
		TeamID:      team.TeamID,
		OwnerUserID: team.OwnerID,
		CreatedAt:   now.UTC().Format(time.RFC3339),
	}, team.TeamID, now)
}

func (s *Service) enqueueTeamMemberAdded(ctx context.Context, teamID, userID, role, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventTeamMemberAdded, traceID, contracts.TeamMemberAddedPayload{
		TeamID:  strings.TrimSpace(teamID),
		UserID:  strings.TrimSpace(userID),
		Role:    strings.TrimSpace(role),
		AddedAt: now.UTC().Format(time.RFC3339),
	}, teamID, now)
}

func (s *Service) enqueueTeamMemberRemoved(ctx context.Context, teamID, userID, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventTeamMemberRemoved, traceID, contracts.TeamMemberRemovedPayload{
		TeamID:    strings.TrimSpace(teamID),
		UserID:    strings.TrimSpace(userID),
		RemovedAt: now.UTC().Format(time.RFC3339),
	}, teamID, now)
}

func (s *Service) enqueueTeamInviteSent(ctx context.Context, invite domain.Invite, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventTeamInviteSent, traceID, contracts.TeamInviteSentPayload{
		TeamID:   invite.TeamID,
		InviteID: invite.InviteID,
		Email:    invite.Email,
		Role:     invite.Role,
		SentAt:   now.UTC().Format(time.RFC3339),
	}, invite.TeamID, now)
}

func (s *Service) enqueueTeamInviteAccepted(ctx context.Context, teamID, inviteID, userID, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventTeamInviteAccepted, traceID, contracts.TeamInviteAcceptedPayload{
		TeamID:     strings.TrimSpace(teamID),
		InviteID:   strings.TrimSpace(inviteID),
		UserID:     strings.TrimSpace(userID),
		AcceptedAt: now.UTC().Format(time.RFC3339),
	}, teamID, now)
}

func (s *Service) enqueueTeamRoleChanged(ctx context.Context, teamID, userID, oldRole, newRole, traceID string, now time.Time) error {
	return s.enqueueEvent(ctx, domain.EventTeamRoleChanged, traceID, contracts.TeamRoleChangedPayload{
		TeamID:    strings.TrimSpace(teamID),
		UserID:    strings.TrimSpace(userID),
		OldRole:   strings.TrimSpace(oldRole),
		NewRole:   strings.TrimSpace(newRole),
		ChangedAt: now.UTC().Format(time.RFC3339),
	}, teamID, now)
}

func publishDLQIdempotencyConflictRecord(key, source, traceID string, now time.Time) contracts.DLQRecord {
	return contracts.DLQRecord{OriginalEvent: contracts.EventEnvelope{EventID: uuid.NewString(), EventType: "team.idempotency.conflict", EventClass: domain.CanonicalEventClassOps, OccurredAt: now, PartitionKeyPath: "envelope.source_service", PartitionKey: source, SourceService: source, TraceID: traceID, SchemaVersion: "v1", Data: []byte(`{"key":"` + key + `"}`)}, ErrorSummary: "idempotency key reused with mismatched payload", RetryCount: 1, FirstSeenAt: now, LastErrorAt: now, SourceTopic: "api", DLQTopic: "team-service.dlq", TraceID: traceID}
}

func (s *Service) publishDLQIdempotencyConflict(ctx context.Context, key, traceID string) error {
	if s.dlq == nil {
		return nil
	}
	now := s.nowFn()
	trace := strings.TrimSpace(traceID)
	if trace == "" {
		trace = uuid.NewString()
	}
	return s.dlq.PublishDLQ(ctx, publishDLQIdempotencyConflictRecord(key, s.cfg.ServiceName, trace, now))
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

func validatePartitionKeyInvariant(event contracts.EventEnvelope) error {
	if event.PartitionKeyPath == "envelope.source_service" {
		if event.PartitionKey != event.SourceService {
			return domain.ErrInvalidEnvelope
		}
		return nil
	}
	if !strings.HasPrefix(event.PartitionKeyPath, "data.") {
		return domain.ErrInvalidEnvelope
	}
	field := strings.TrimPrefix(event.PartitionKeyPath, "data.")
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
