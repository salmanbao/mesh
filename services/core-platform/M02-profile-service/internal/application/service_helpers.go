package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

type profileUpdatedEventData struct {
	UserID          string `json:"user_id"`
	UpdatedAt       string `json:"updated_at"`
	Username        string `json:"username,omitempty"`
	DisplayName     string `json:"display_name,omitempty"`
	Bio             string `json:"bio,omitempty"`
	IsPrivate       bool   `json:"is_private,omitempty"`
	IsUnlisted      bool   `json:"is_unlisted,omitempty"`
	AnalyticsOptOut bool   `json:"analytics_opt_out,omitempty"`
}

func (s *Service) enqueueProfileUpdated(ctx context.Context, profile domain.Profile) error {
	occurredAt := s.nowFn()
	data := profileUpdatedEventData{
		UserID:          profile.UserID.String(),
		UpdatedAt:       occurredAt.Format(time.RFC3339),
		Username:        profile.Username,
		DisplayName:     profile.DisplayName,
		Bio:             profile.Bio,
		IsPrivate:       profile.IsPrivate,
		IsUnlisted:      profile.IsUnlisted,
		AnalyticsOptOut: profile.AnalyticsOptOut,
	}
	payloadEnvelope := map[string]any{
		"event_id":           uuid.NewString(),
		"event_type":         "user.profile_updated",
		"occurred_at":        occurredAt.Format(time.RFC3339),
		"source_service":     s.cfg.ServiceName,
		"trace_id":           "",
		"schema_version":     "1.0",
		"partition_key_path": "data.user_id",
		"partition_key":      profile.UserID.String(),
		"data":               data,
	}
	payload, _ := json.Marshal(payloadEnvelope)
	return s.outbox.Enqueue(ctx, ports.OutboxEvent{
		EventID:          uuid.New(),
		EventType:        "user.profile_updated",
		PartitionKey:     profile.UserID.String(),
		PartitionKeyPath: "data.user_id",
		Payload:          payload,
		OccurredAt:       occurredAt,
		SchemaVersion:    "1.0",
	})
}

func hashRequest(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) reserveIdempotency(ctx context.Context, key string, request any) error {
	if key == "" {
		return nil
	}
	err := s.idempotency.Reserve(ctx, key, hashRequest(request), s.nowFn().Add(s.cfg.IdempotencyTTL))
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrIdempotencyConflict, err)
	}
	return nil
}
