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

type CreateExportRequest struct {
	UserID string `json:"user_id"`
	Format string `json:"format,omitempty"`
}

type EraseRequest struct {
	UserID string `json:"user_id"`
	Reason string `json:"reason"`
}

type ExportRequestResponse struct {
	RequestID    string `json:"request_id"`
	UserID       string `json:"user_id"`
	RequestType  string `json:"request_type"`
	Format       string `json:"format,omitempty"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	RequestedAt  string `json:"requested_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	DownloadURL  string `json:"download_url,omitempty"`
	FailureCause string `json:"failure_cause,omitempty"`
}

type ExportHistoryResponse struct {
	Items []ExportRequestResponse `json:"items"`
}
