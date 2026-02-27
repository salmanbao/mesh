package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ConnectIntegrationRequest struct {
	Platform      string            `json:"platform"`
	CommunityName string            `json:"community_name"`
	Config        map[string]string `json:"config"`
}

type CommunityIntegrationResponse struct {
	IntegrationID string            `json:"integration_id"`
	CreatorID     string            `json:"creator_id"`
	Platform      string            `json:"platform"`
	Status        string            `json:"status"`
	CommunityName string            `json:"community_name"`
	Config        map[string]string `json:"config,omitempty"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

type ManualGrantRequest struct {
	UserID        string `json:"user_id"`
	ProductID     string `json:"product_id"`
	IntegrationID string `json:"integration_id"`
	Reason        string `json:"reason"`
	Tier          string `json:"tier,omitempty"`
}

type CommunityGrantResponse struct {
	GrantID       string `json:"grant_id"`
	Status        string `json:"status"`
	UserID        string `json:"user_id"`
	ProductID     string `json:"product_id"`
	IntegrationID string `json:"integration_id"`
	Tier          string `json:"tier"`
	GrantedAt     string `json:"granted_at"`
}

type AuditLogResponse struct {
	AuditLogID    string            `json:"audit_log_id"`
	Timestamp     string            `json:"timestamp"`
	ActionType    string            `json:"action_type"`
	PerformedBy   string            `json:"performed_by,omitempty"`
	PerformerRole string            `json:"performer_role,omitempty"`
	UserID        string            `json:"user_id,omitempty"`
	IntegrationID string            `json:"integration_id,omitempty"`
	ProductID     string            `json:"product_id,omitempty"`
	GrantID       string            `json:"grant_id,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	Outcome       string            `json:"outcome"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type AuditLogListResponse struct {
	Items []AuditLogResponse `json:"items"`
}

type HealthCheckResponse struct {
	HealthCheckID  string `json:"health_check_id"`
	IntegrationID  string `json:"integration_id"`
	Status         string `json:"status"`
	CheckedAt      string `json:"checked_at"`
	LatencyMS      int    `json:"latency_ms"`
	HTTPStatusCode int    `json:"http_status_code,omitempty"`
}
