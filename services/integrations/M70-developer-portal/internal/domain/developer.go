package domain

import "time"

const (
	DeveloperStatusActive = "active"

	SessionStatusActive = "active"

	APIKeyStatusActive     = "active"
	APIKeyStatusDeprecated = "deprecated"
	APIKeyStatusRevoked    = "revoked"

	WebhookStatusActive = "active"

	DeliveryStatusSuccess = "success"
)

type Developer struct {
	DeveloperID string    `json:"developer_id"`
	Email       string    `json:"email"`
	AppName     string    `json:"app_name"`
	Tier        string    `json:"tier"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type DeveloperSession struct {
	SessionID    string    `json:"session_id"`
	DeveloperID  string    `json:"developer_id"`
	SessionToken string    `json:"session_token"`
	Status       string    `json:"status"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	KeyID       string     `json:"key_id"`
	DeveloperID string     `json:"developer_id"`
	Label       string     `json:"label"`
	MaskedKey   string     `json:"masked_key"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

type APIKeyRotation struct {
	RotationID  string    `json:"rotation_id"`
	OldKeyID    string    `json:"old_key_id"`
	NewKeyID    string    `json:"new_key_id"`
	DeveloperID string    `json:"developer_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type Webhook struct {
	WebhookID   string    `json:"webhook_id"`
	DeveloperID string    `json:"developer_id"`
	URL         string    `json:"url"`
	EventType   string    `json:"event_type"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type WebhookDelivery struct {
	DeliveryID string    `json:"delivery_id"`
	WebhookID  string    `json:"webhook_id"`
	Status     string    `json:"status"`
	TestEvent  bool      `json:"test_event"`
	CreatedAt  time.Time `json:"created_at"`
}

type DeveloperUsage struct {
	UsageID      string    `json:"usage_id"`
	DeveloperID  string    `json:"developer_id"`
	CurrentUsage int       `json:"current_usage"`
	RateLimit    int       `json:"rate_limit"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
}

type AuditLog struct {
	AuditID     string            `json:"audit_id"`
	DeveloperID string            `json:"developer_id"`
	ActionType  string            `json:"action_type"`
	EntityID    string            `json:"entity_id"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
