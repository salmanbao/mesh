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

type RecommendationGeneratedPayload struct {
	RecommendationID      string  `json:"recommendation_id,omitempty"`
	RecommendationBatchID string  `json:"recommendation_batch_id,omitempty"`
	EntityID              string  `json:"entity_id"`
	ModelVersion          string  `json:"model_version"`
	RecommendationScore   float64 `json:"recommendation_score"`
	UserID                string  `json:"user_id"`
}

type RecommendationFeedbackRecordedPayload struct {
	FeedbackID        string `json:"feedback_id"`
	RecommendationID  string `json:"recommendation_id"`
	EntityID          string `json:"entity_id"`
	FeedbackEventType string `json:"feedback_event_type"`
	ModelVersion      string `json:"model_version,omitempty"`
	UserID            string `json:"user_id"`
}

type RecommendationOverrideAppliedPayload struct {
	OverrideID    string  `json:"override_id"`
	EntityID      string  `json:"entity_id"`
	Multiplier    float64 `json:"multiplier"`
	AffectedUsers int     `json:"affected_users"`
	OverrideType  string  `json:"override_type,omitempty"`
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
