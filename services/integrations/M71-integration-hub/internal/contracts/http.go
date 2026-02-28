package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status    string       `json:"status"`
	Code      string       `json:"code,omitempty"`
	Message   string       `json:"message,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
	Error     ErrorPayload `json:"error"`
}

type AuthorizeIntegrationRequest struct {
	UserID          string `json:"user_id,omitempty"`
	IntegrationName string `json:"integration_name,omitempty"`
}

type CreateWorkflowRequest struct {
	UserID              string `json:"user_id,omitempty"`
	WorkflowName        string `json:"workflow_name"`
	WorkflowDescription string `json:"workflow_description,omitempty"`
	TriggerEventType    string `json:"trigger_event_type"`
	ActionType          string `json:"action_type"`
	IntegrationID       string `json:"integration_id,omitempty"`
}

type CreateWebhookRequest struct {
	UserID      string `json:"user_id,omitempty"`
	EndpointURL string `json:"endpoint_url"`
	EventType   string `json:"event_type"`
}

type IntegrationResponse struct {
	IntegrationID   string `json:"integration_id"`
	UserID          string `json:"user_id"`
	IntegrationType string `json:"integration_type"`
	IntegrationName string `json:"integration_name"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
}

type WorkflowResponse struct {
	WorkflowID          string `json:"workflow_id"`
	UserID              string `json:"user_id"`
	WorkflowName        string `json:"workflow_name"`
	WorkflowDescription string `json:"workflow_description,omitempty"`
	TriggerEventType    string `json:"trigger_event_type"`
	ActionType          string `json:"action_type"`
	IntegrationID       string `json:"integration_id,omitempty"`
	Status              string `json:"status"`
	CreatedAt           string `json:"created_at"`
}

type WorkflowExecutionResponse struct {
	ExecutionID string `json:"execution_id"`
	WorkflowID  string `json:"workflow_id"`
	Status      string `json:"status"`
	TestRun     bool   `json:"test_run"`
	StartedAt   string `json:"started_at"`
}

type WebhookResponse struct {
	WebhookID   string `json:"webhook_id"`
	UserID      string `json:"user_id"`
	EndpointURL string `json:"endpoint_url"`
	EventType   string `json:"event_type"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type WebhookDeliveryResponse struct {
	DeliveryID string `json:"delivery_id"`
	WebhookID  string `json:"webhook_id"`
	Status     string `json:"status"`
	TestEvent  bool   `json:"test_event"`
	CreatedAt  string `json:"created_at"`
}

type ChatPostMessageResponse struct {
	Channel   string `json:"channel"`
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}
