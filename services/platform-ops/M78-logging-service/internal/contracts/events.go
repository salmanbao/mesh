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

type LoggingAlertTriggered struct {
	Service   string  `json:"service"`
	Metric    string  `json:"metric"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Severity  string  `json:"severity"`
	RuleID    string  `json:"rule_id"`
	Message   string  `json:"message,omitempty"`
	TraceID   string  `json:"trace_id,omitempty"`
}
