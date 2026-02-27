package contracts

type CreateExportRequest struct {
	ExportType string            `json:"export_type"`
	Period     string            `json:"period"`
	Format     string            `json:"format"`
	Filters    map[string]string `json:"filters"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}
