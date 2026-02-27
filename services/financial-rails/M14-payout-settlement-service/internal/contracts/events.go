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

type RewardPayoutEligiblePayload struct {
	SubmissionID string  `json:"submission_id"`
	UserID       string  `json:"user_id"`
	CampaignID   string  `json:"campaign_id"`
	LockedViews  int64   `json:"locked_views"`
	RatePer1K    float64 `json:"rate_per_1k"`
	GrossAmount  float64 `json:"gross_amount"`
	EligibleAt   string  `json:"eligible_at"`
}

type PayoutProcessingPayload struct {
	PayoutID     string  `json:"payout_id"`
	UserID       string  `json:"user_id"`
	Amount       float64 `json:"amount"`
	Method       string  `json:"method"`
	ProcessingAt string  `json:"processing_at"`
}

type PayoutPaidPayload struct {
	PayoutID string  `json:"payout_id"`
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Method   string  `json:"method"`
	PaidAt   string  `json:"paid_at"`
}

type PayoutFailedPayload struct {
	PayoutID string  `json:"payout_id"`
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Method   string  `json:"method"`
	FailedAt string  `json:"failed_at"`
	Reason   string  `json:"reason"`
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
