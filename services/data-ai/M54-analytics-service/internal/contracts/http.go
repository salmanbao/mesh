package contracts

type ExportRequest struct {
	ReportType string            `json:"report_type"`
	Format     string            `json:"format"`
	DateFrom   string            `json:"date_from"`
	DateTo     string            `json:"date_to"`
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
