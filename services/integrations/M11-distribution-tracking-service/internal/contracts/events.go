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

type TrackingMetricsUpdatedPayload struct {
	TrackedPostID string `json:"tracked_post_id"`
	Platform      string `json:"platform"`
	Views         int    `json:"views"`
	Likes         int    `json:"likes"`
	Shares        int    `json:"shares"`
	Comments      int    `json:"comments"`
	PolledAt      string `json:"polled_at"`
}

type TrackingPostArchivedPayload struct {
	TrackedPostID string `json:"tracked_post_id"`
	ArchivedAt    string `json:"archived_at"`
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
