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

type ChargebackWebhookData struct {
	Amount        float64 `json:"amount"`
	ChargeID      string  `json:"charge_id"`
	Currency      string  `json:"currency"`
	DisputeReason string  `json:"dispute_reason"`
	SellerID      string  `json:"seller_id"`
}

type ChargebackWebhookEnvelope struct {
	EventID          string                `json:"event_id"`
	EventType        string                `json:"event_type"`
	OccurredAt       string                `json:"occurred_at"`
	SourceService    string                `json:"source_service"`
	TraceID          string                `json:"trace_id"`
	SchemaVersion    string                `json:"schema_version"`
	PartitionKeyPath string                `json:"partition_key_path"`
	PartitionKey     string                `json:"partition_key"`
	Data             ChargebackWebhookData `json:"data"`
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
