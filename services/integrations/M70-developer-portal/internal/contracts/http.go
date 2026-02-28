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

type RegisterDeveloperRequest struct {
	Email   string `json:"email"`
	AppName string `json:"app_name"`
}

type CreateAPIKeyRequest struct {
	DeveloperID string `json:"developer_id,omitempty"`
	Label       string `json:"label"`
}

type CreateWebhookRequest struct {
	DeveloperID string `json:"developer_id,omitempty"`
	URL         string `json:"url"`
	EventType   string `json:"event_type"`
}

type DeveloperResponse struct {
	DeveloperID string `json:"developer_id"`
	Email       string `json:"email"`
	AppName     string `json:"app_name"`
	Tier        string `json:"tier"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type SessionResponse struct {
	SessionID    string `json:"session_id"`
	DeveloperID  string `json:"developer_id"`
	SessionToken string `json:"session_token"`
	Status       string `json:"status"`
	ExpiresAt    string `json:"expires_at"`
	CreatedAt    string `json:"created_at"`
}

type RegisterDeveloperResponse struct {
	Developer DeveloperResponse `json:"developer"`
	Session   SessionResponse   `json:"session"`
}

type APIKeyResponse struct {
	KeyID       string `json:"key_id"`
	DeveloperID string `json:"developer_id"`
	Label       string `json:"label"`
	MaskedKey   string `json:"masked_key"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	RevokedAt   string `json:"revoked_at,omitempty"`
}

type APIKeyRotationResponse struct {
	RotationID  string         `json:"rotation_id"`
	OldKey      APIKeyResponse `json:"old_key"`
	NewKey      APIKeyResponse `json:"new_key"`
	DeveloperID string         `json:"developer_id"`
	CreatedAt   string         `json:"created_at"`
}

type WebhookResponse struct {
	WebhookID   string `json:"webhook_id"`
	DeveloperID string `json:"developer_id"`
	URL         string `json:"url"`
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
