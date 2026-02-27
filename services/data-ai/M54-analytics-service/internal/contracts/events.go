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

type SubmissionPayload struct {
	SubmissionID string `json:"submission_id"`
	UserID       string `json:"user_id"`
	CampaignID   string `json:"campaign_id"`
	Platform     string `json:"platform"`
	Status       string `json:"status,omitempty"`
	Views        int64  `json:"views,omitempty"`
	OccurredAt   string `json:"occurred_at,omitempty"`
}

type PayoutPaidPayload struct {
	PayoutID   string  `json:"payout_id"`
	UserID     string  `json:"user_id"`
	Amount     float64 `json:"amount"`
	OccurredAt string  `json:"occurred_at"`
}

type RewardCalculatedPayload struct {
	SubmissionID string  `json:"submission_id"`
	UserID       string  `json:"user_id"`
	GrossAmount  float64 `json:"gross_amount"`
	NetAmount    float64 `json:"net_amount"`
	CalculatedAt string  `json:"calculated_at"`
}

type CampaignLaunchedPayload struct {
	CampaignID string  `json:"campaign_id"`
	BrandID    string  `json:"brand_id"`
	Category   string  `json:"category"`
	RewardRate float64 `json:"reward_rate"`
	Budget     float64 `json:"budget"`
	LaunchedAt string  `json:"launched_at"`
}

type UserRegisteredPayload struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	Country   string `json:"country"`
	CreatedAt string `json:"created_at"`
}

type TransactionPayload struct {
	TransactionID string  `json:"transaction_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	OccurredAt    string  `json:"occurred_at"`
	Reason        string  `json:"reason,omitempty"`
}

type ClickPayload struct {
	ClickID       string `json:"click_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	Platform      string `json:"platform,omitempty"`
	ItemType      string `json:"item_type,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	DownloadID    string `json:"download_id,omitempty"`
	TrackedPostID string `json:"tracked_post_id,omitempty"`
	OccurredAt    string `json:"occurred_at,omitempty"`
}

type ConsentPayload struct {
	UserID    string `json:"user_id"`
	Analytics bool   `json:"analytics"`
	UpdatedAt string `json:"updated_at,omitempty"`
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
