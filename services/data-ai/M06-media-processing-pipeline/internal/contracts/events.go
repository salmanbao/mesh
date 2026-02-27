package contracts

import "time"

type QueueDLQRecord struct {
	JobID        string    `json:"job_id"`
	AssetID      string    `json:"asset_id"`
	JobType      string    `json:"job_type"`
	ErrorSummary string    `json:"error_summary"`
	RetryCount   int       `json:"retry_count"`
	FirstSeenAt  time.Time `json:"first_seen_at"`
	LastErrorAt  time.Time `json:"last_error_at"`
}
