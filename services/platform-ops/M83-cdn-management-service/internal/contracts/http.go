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

type CreateConfigRequest struct {
	Provider     string            `json:"provider"`
	Config       map[string]any    `json:"config"`
	HeaderRules  map[string]string `json:"header_rules,omitempty"`
	SignedURLTTL int               `json:"signed_url_ttl_seconds,omitempty"`
}

type PurgeRequest struct {
	Scope  string `json:"scope"`
	Target string `json:"target"`
}
