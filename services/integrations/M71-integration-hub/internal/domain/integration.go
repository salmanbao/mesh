package domain

import "time"

const (
	IntegrationStatusConnected = "connected"

	WebhookStatusActive = "active"

	WorkflowStatusDraft     = "draft"
	WorkflowStatusPublished = "published"

	ExecutionStatusSuccess = "success"
)

type Integration struct {
	IntegrationID   string    `json:"integration_id"`
	UserID          string    `json:"user_id"`
	IntegrationType string    `json:"integration_type"`
	IntegrationName string    `json:"integration_name"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type APICredential struct {
	CredentialID   string    `json:"credential_id"`
	IntegrationID  string    `json:"integration_id"`
	CredentialType string    `json:"credential_type"`
	MaskedValue    string    `json:"masked_value"`
	CreatedAt      time.Time `json:"created_at"`
}

type Workflow struct {
	WorkflowID          string    `json:"workflow_id"`
	UserID              string    `json:"user_id"`
	WorkflowName        string    `json:"workflow_name"`
	WorkflowDescription string    `json:"workflow_description"`
	TriggerEventType    string    `json:"trigger_event_type"`
	ActionType          string    `json:"action_type"`
	IntegrationID       string    `json:"integration_id,omitempty"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

type WorkflowExecution struct {
	ExecutionID string    `json:"execution_id"`
	WorkflowID  string    `json:"workflow_id"`
	Status      string    `json:"status"`
	TestRun     bool      `json:"test_run"`
	StartedAt   time.Time `json:"started_at"`
}

type Webhook struct {
	WebhookID   string    `json:"webhook_id"`
	UserID      string    `json:"user_id"`
	EndpointURL string    `json:"endpoint_url"`
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

type Analytics struct {
	AnalyticsID      string    `json:"analytics_id"`
	IntegrationID    string    `json:"integration_id"`
	TotalEvents      int       `json:"total_events"`
	FailedEvents     int       `json:"failed_events"`
	AggregationStart time.Time `json:"aggregation_start"`
}

type IntegrationLog struct {
	LogID         string    `json:"log_id"`
	IntegrationID string    `json:"integration_id"`
	ActionType    string    `json:"action_type"`
	Status        string    `json:"status"`
	ActionAt      time.Time `json:"action_at"`
}
