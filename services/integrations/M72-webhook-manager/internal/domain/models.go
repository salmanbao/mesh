package domain

import (
	"time"
)

type Webhook struct {
	WebhookID           string    `json:"webhook_id"`
	EndpointURL         string    `json:"endpoint_url"`
	EventTypes          []string  `json:"event_types"`
	Status              string    `json:"status"`
	SigningSecret       string    `json:"signing_secret,omitempty"`
	BatchModeEnabled    bool      `json:"batch_mode_enabled"`
	BatchSize           int       `json:"batch_size"`
	BatchWindowSeconds  int       `json:"batch_window_seconds"`
	RateLimitPerMinute  int       `json:"rate_limit_per_minute"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Delivery struct {
	DeliveryID      string    `json:"delivery_id"`
	WebhookID       string    `json:"webhook_id"`
	OriginalEventID string    `json:"original_event_id"`
	OriginalType    string    `json:"original_event_type"`
	HTTPStatus      int       `json:"http_status"`
	LatencyMS       int64     `json:"latency_ms"`
	RetryCount      int       `json:"retry_count"`
	DeliveredAt     time.Time `json:"delivered_at"`
	IsTest          bool      `json:"is_test"`
	Success         bool      `json:"success"`
}

type Analytics struct {
	TotalDeliveries      int64              `json:"total_deliveries"`
	SuccessfulDeliveries int64              `json:"successful_deliveries"`
	FailedDeliveries     int64              `json:"failed_deliveries"`
	SuccessRate          float64            `json:"success_rate"`
	AvgLatencyMS         float64            `json:"avg_latency_ms"`
	P95LatencyMS         float64            `json:"p95_latency_ms"`
	P99LatencyMS         float64            `json:"p99_latency_ms"`
	ByEventType          map[string]Metrics `json:"by_event_type"`
}

type Metrics struct {
	Total      int64   `json:"total"`
	Success    int64   `json:"success"`
	Failed     int64   `json:"failed"`
	AvgLatency float64 `json:"avg_latency"`
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Response    []byte
	ExpiresAt   time.Time
}
