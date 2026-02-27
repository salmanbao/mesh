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

type SubmissionApprovedPayload struct {
	SubmissionID string `json:"submission_id"`
	UserID       string `json:"user_id"`
	CampaignID   string `json:"campaign_id"`
	ApprovedAt   string `json:"approved_at"`
}

type PayoutFailedPayload struct {
	PayoutID string  `json:"payout_id"`
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Method   string  `json:"method"`
	FailedAt string  `json:"failed_at"`
	Reason   string  `json:"reason"`
}

type DisputeCreatedPayload struct {
	DisputeID  string `json:"dispute_id"`
	UserID     string `json:"user_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	CreatedAt  string `json:"created_at"`
}

type DisputeResolvedPayload struct {
	DisputeID  string `json:"dispute_id"`
	ResolvedAt string `json:"resolved_at"`
	Resolution string `json:"resolution"`
}

type TransactionRefundedPayload struct {
	TransactionID string  `json:"transaction_id"`
	RefundID      string  `json:"refund_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency,omitempty"`
	Provider      string  `json:"provider,omitempty"`
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
	DLQTopic      string        `json:"dlq_topic,omitempty"`
	TraceID       string        `json:"trace_id,omitempty"`
}
