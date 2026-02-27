package contracts

import (
	"encoding/json"
	"time"
)

type EventEnvelope struct {
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	EventClass       string          `json:"event_class"`
	OccurredAt       time.Time       `json:"occurred_at"`
	PartitionKey     string          `json:"partition_key"`
	PartitionKeyPath string          `json:"partition_key_path"`
	SourceService    string          `json:"source_service"`
	TraceID          string          `json:"trace_id"`
	SchemaVersion    string          `json:"schema_version"`
	Data             json.RawMessage `json:"data"`
}

type PayoutPaidPayload struct {
	PayoutID    string  `json:"payout_id"`
	UserID      string  `json:"user_id"`
	CreatorID   string  `json:"creator_id,omitempty"`
	Amount      float64 `json:"amount"`
	GrossAmount float64 `json:"gross_amount,omitempty"`
	FeeAmount   float64 `json:"fee_amount,omitempty"`
	NetAmount   float64 `json:"net_amount,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	Method      string  `json:"method,omitempty"`
	PaidAt      string  `json:"paid_at"`
}

type PayoutFailedPayload struct {
	PayoutID   string  `json:"payout_id"`
	UserID     string  `json:"user_id"`
	CreatorID  string  `json:"creator_id,omitempty"`
	Amount     float64 `json:"amount,omitempty"`
	Method     string  `json:"method,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	ReasonCode string  `json:"reason_code,omitempty"`
	FailedAt   string  `json:"failed_at"`
}

type DLQRecord struct {
	OriginalEvent EventEnvelope `json:"original_event"`
	ErrorSummary  string        `json:"error_summary"`
	RetryCount    int           `json:"retry_count"`
	FirstSeenAt   time.Time     `json:"first_seen_at"`
	LastErrorAt   time.Time     `json:"last_error_at"`
	SourceTopic   string        `json:"source_topic,omitempty"`
	TraceID       string        `json:"trace_id,omitempty"`
}
