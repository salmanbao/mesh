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

type TransactionSucceededPayload struct {
	TransactionID string  `json:"transaction_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Provider      string  `json:"provider"`
	OccurredAt    string  `json:"occurred_at"`
}

type TransactionFailedPayload struct {
	TransactionID string  `json:"transaction_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Provider      string  `json:"provider"`
	OccurredAt    string  `json:"occurred_at"`
	Reason        string  `json:"reason"`
}

type TransactionRefundedPayload struct {
	TransactionID string  `json:"transaction_id"`
	RefundID      string  `json:"refund_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Provider      string  `json:"provider"`
	OccurredAt    string  `json:"occurred_at"`
	Reason        string  `json:"reason"`
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
