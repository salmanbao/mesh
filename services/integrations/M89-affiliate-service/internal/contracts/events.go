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

type AffiliateClickTrackedPayload struct {
	AffiliateID string `json:"affiliate_id"`
	LinkID      string `json:"link_id"`
	ReferrerURL string `json:"referrer_url"`
	IPHash      string `json:"ip_hash"`
	TrackedAt   string `json:"tracked_at"`
}

type AffiliateAttributionCreatedPayload struct {
	AffiliateID  string  `json:"affiliate_id"`
	ConversionID string  `json:"conversion_id"`
	OrderID      string  `json:"order_id"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency,omitempty"`
	AttributedAt string  `json:"attributed_at"`
}

type AffiliateLinkCreatedPayload struct {
	AffiliateID string `json:"affiliate_id"`
	LinkID      string `json:"link_id"`
	Token       string `json:"token"`
	Channel     string `json:"channel,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type AffiliateEarningCalculatedPayload struct {
	AffiliateID   string  `json:"affiliate_id"`
	EarningID     string  `json:"earning_id"`
	AttributionID string  `json:"attribution_id"`
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency,omitempty"`
	Status        string  `json:"status"`
	CalculatedAt  string  `json:"calculated_at"`
}

type AffiliatePayoutQueuedPayload struct {
	AffiliateID string  `json:"affiliate_id"`
	PayoutID    string  `json:"payout_id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	QueuedAt    string  `json:"queued_at"`
}
