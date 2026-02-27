package contracts

import (
	"encoding/json"
	"time"
)

type EventEnvelope struct {
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	EventClass       string          `json:"event_class,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
	PartitionKeyPath string          `json:"partition_key_path"`
	PartitionKey     string          `json:"partition_key"`
	SourceService    string          `json:"source_service"`
	TraceID          string          `json:"trace_id"`
	SchemaVersion    string          `json:"schema_version"`
	Data             json.RawMessage `json:"data"`
}

type TeamCreatedPayload struct {
	TeamID      string `json:"team_id"`
	OwnerUserID string `json:"owner_user_id"`
	TeamName    string `json:"team_name,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type TeamMemberAddedPayload struct {
	TeamID  string `json:"team_id"`
	UserID  string `json:"user_id"`
	Role    string `json:"role"`
	AddedAt string `json:"added_at"`
}

type TeamMemberRemovedPayload struct {
	TeamID    string `json:"team_id"`
	UserID    string `json:"user_id"`
	RemovedAt string `json:"removed_at"`
}

type TeamInviteSentPayload struct {
	TeamID   string `json:"team_id"`
	InviteID string `json:"invite_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	SentAt   string `json:"sent_at"`
}

type TeamInviteAcceptedPayload struct {
	TeamID     string `json:"team_id"`
	InviteID   string `json:"invite_id"`
	UserID     string `json:"user_id"`
	AcceptedAt string `json:"accepted_at"`
}

type TeamRoleChangedPayload struct {
	TeamID    string `json:"team_id"`
	UserID    string `json:"user_id"`
	OldRole   string `json:"old_role"`
	NewRole   string `json:"new_role"`
	ChangedAt string `json:"changed_at"`
}

type DLQRecord struct {
	OriginalEvent EventEnvelope `json:"original_event"`
	ErrorSummary  string        `json:"error_summary"`
	RetryCount    int           `json:"retry_count"`
	FirstSeenAt   time.Time     `json:"first_seen_at"`
	LastErrorAt   time.Time     `json:"last_error_at"`
	SourceTopic   string        `json:"source_topic,omitempty"`
	DLQTopic      string        `json:"dlq_topic,omitempty"`
	TraceID       string        `json:"trace_id,omitempty"`
}
