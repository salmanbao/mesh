package domain

import "time"

const (
	ConsentStatusNoConsent = "no_consent"
	ConsentStatusActive    = "active"
	ConsentStatusWithdrawn = "withdrawn"
)

type ConsentRecord struct {
	UserID      string          `json:"user_id"`
	Preferences map[string]bool `json:"preferences"`
	Status      string          `json:"status"`
	UpdatedAt   time.Time       `json:"updated_at"`
	UpdatedBy   string          `json:"updated_by"`
	LastReason  string          `json:"last_reason,omitempty"`
}

type ConsentHistory struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	UserID     string    `json:"user_id"`
	Category   string    `json:"category,omitempty"`
	Reason     string    `json:"reason"`
	ChangedBy  string    `json:"changed_by"`
	OccurredAt time.Time `json:"occurred_at"`
}

type AuditLog struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type"`
	UserID     string            `json:"user_id"`
	ActorID    string            `json:"actor_id"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
