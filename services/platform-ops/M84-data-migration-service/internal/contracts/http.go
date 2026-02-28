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

type CreatePlanRequest struct {
	ServiceName string         `json:"service_name"`
	Environment string         `json:"environment"`
	Version     string         `json:"version"`
	Plan        map[string]any `json:"plan"`
	DryRun      bool           `json:"dry_run,omitempty"`
	RiskLevel   string         `json:"risk_level,omitempty"`
}

type CreateRunRequest struct {
	PlanID string `json:"plan_id"`
}
