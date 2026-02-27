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

type SubmissionAutoApprovedPayload struct {
	SubmissionID   string `json:"submission_id"`
	UserID         string `json:"user_id"`
	CampaignID     string `json:"campaign_id"`
	AutoApprovedAt string `json:"auto_approved_at"`
}

type SubmissionVerifiedPayload struct {
	SubmissionID string `json:"submission_id"`
	UserID       string `json:"user_id"`
	CampaignID   string `json:"campaign_id"`
	VerifiedAt   string `json:"verified_at"`
}

type SubmissionViewLockedPayload struct {
	SubmissionID string `json:"submission_id"`
	UserID       string `json:"user_id"`
	CampaignID   string `json:"campaign_id"`
	LockedViews  int64  `json:"locked_views"`
	LockedAt     string `json:"locked_at"`
}

type SubmissionCancelledPayload struct {
	SubmissionID string `json:"submission_id"`
	UserID       string `json:"user_id"`
	CampaignID   string `json:"campaign_id"`
	CancelledAt  string `json:"cancelled_at"`
	Reason       string `json:"reason"`
}

type TrackingMetricsUpdatedPayload struct {
	TrackedPostID string `json:"tracked_post_id"`
	Platform      string `json:"platform"`
	Views         int64  `json:"views"`
	Likes         int64  `json:"likes"`
	Shares        int64  `json:"shares"`
	Comments      int64  `json:"comments"`
	PolledAt      string `json:"polled_at"`
}

type RewardCalculatedPayload struct {
	SubmissionID            string  `json:"submission_id"`
	UserID                  string  `json:"user_id"`
	CampaignID              string  `json:"campaign_id"`
	LockedViews             int64   `json:"locked_views"`
	RatePer1K               float64 `json:"rate_per_1k"`
	GrossAmount             float64 `json:"gross_amount"`
	NetAmount               float64 `json:"net_amount"`
	RolloverApplied         float64 `json:"rollover_applied"`
	RolloverBalance         float64 `json:"rollover_balance"`
	VerificationCompletedAt string  `json:"verification_completed_at"`
	CalculatedAt            string  `json:"calculated_at"`
	Status                  string  `json:"status"`
	FraudScore              float64 `json:"fraud_score,omitempty"`
}

type RewardPayoutEligiblePayload struct {
	SubmissionID            string  `json:"submission_id"`
	UserID                  string  `json:"user_id"`
	CampaignID              string  `json:"campaign_id"`
	LockedViews             int64   `json:"locked_views"`
	RatePer1K               float64 `json:"rate_per_1k"`
	GrossAmount             float64 `json:"gross_amount"`
	NetAmount               float64 `json:"net_amount"`
	RolloverApplied         float64 `json:"rollover_applied"`
	RolloverBalance         float64 `json:"rollover_balance"`
	EligibleAt              string  `json:"eligible_at"`
	VerificationCompletedAt string  `json:"verification_completed_at"`
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
