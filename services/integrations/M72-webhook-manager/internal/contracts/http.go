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

type CreateWebhookRequest struct {
	EndpointURL        string   `json:"endpoint_url"`
	EventTypes         []string `json:"event_types"`
	BatchModeEnabled   bool     `json:"batch_mode_enabled"`
	BatchSize          int      `json:"batch_size"`
	BatchWindowSeconds int      `json:"batch_window_seconds"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute"`
}

type UpdateWebhookRequest struct {
	EventTypes         []string `json:"event_types,omitempty"`
	BatchModeEnabled   *bool    `json:"batch_mode_enabled,omitempty"`
	BatchSize          int      `json:"batch_size,omitempty"`
	BatchWindowSeconds int      `json:"batch_window_seconds,omitempty"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty"`
	Status             string   `json:"status,omitempty"`
}

type TestWebhookRequest struct {
	Payload any `json:"payload,omitempty"`
}

type InboundWebhookResponse struct {
	Accepted bool   `json:"accepted"`
	EventID  string `json:"event_id,omitempty"`
	Size     int    `json:"size"`
}
