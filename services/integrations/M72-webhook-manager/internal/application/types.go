package application

import (
	"time"
)

type Config struct {
	ServiceName    string
	Version        string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateWebhookInput struct {
	EndpointURL        string   `json:"endpoint_url"`
	EventTypes         []string `json:"event_types"`
	BatchModeEnabled   bool     `json:"batch_mode_enabled"`
	BatchSize          int      `json:"batch_size"`
	BatchWindowSeconds int      `json:"batch_window_seconds"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute"`
}

type TestResult struct {
	WebhookID  string    `json:"webhook_id"`
	Status     string    `json:"status"`
	HTTPStatus int       `json:"http_status"`
	LatencyMS  int64     `json:"latency_ms"`
	Timestamp  time.Time `json:"timestamp"`
}
