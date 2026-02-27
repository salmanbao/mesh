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

type EscrowHoldCreatedPayload struct {
	EscrowID   string  `json:"escrow_id"`
	CampaignID string  `json:"campaign_id"`
	CreatorID  string  `json:"creator_id"`
	Amount     float64 `json:"amount"`
	HeldAt     string  `json:"held_at"`
}

type EscrowPartialReleasePayload struct {
	EscrowID          string  `json:"escrow_id"`
	Amount            float64 `json:"amount"`
	RemainingBalance  float64 `json:"remaining_balance"`
	ReleasedAt        string  `json:"released_at"`
}

type EscrowHoldFullyReleasedPayload struct {
	EscrowID    string `json:"escrow_id"`
	ReleasedAt  string `json:"released_at"`
}

type EscrowRefundProcessedPayload struct {
	EscrowID    string  `json:"escrow_id"`
	Amount      float64 `json:"amount"`
	RefundedAt  string  `json:"refunded_at"`
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
