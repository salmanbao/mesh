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

type AnalyzeRequest struct {
	UserID       string `json:"user_id,omitempty"`
	ContentID    string `json:"content_id,omitempty"`
	Content      string `json:"content"`
	ModelID      string `json:"model_id,omitempty"`
	ModelVersion string `json:"model_version,omitempty"`
}

type BatchItemRequest struct {
	ContentID string `json:"content_id,omitempty"`
	Content   string `json:"content"`
}

type BatchAnalyzeRequest struct {
	UserID       string             `json:"user_id,omitempty"`
	ModelID      string             `json:"model_id,omitempty"`
	ModelVersion string             `json:"model_version,omitempty"`
	Items        []BatchItemRequest `json:"items"`
}

type PredictionResponse struct {
	PredictionID string  `json:"prediction_id"`
	UserID       string  `json:"user_id"`
	ContentID    string  `json:"content_id,omitempty"`
	Label        string  `json:"label"`
	Confidence   float64 `json:"confidence"`
	Flagged      bool    `json:"flagged"`
	ModelID      string  `json:"model_id"`
	ModelVersion string  `json:"model_version"`
	CreatedAt    string  `json:"created_at"`
}

type BatchStatusResponse struct {
	JobID          string               `json:"job_id"`
	UserID         string               `json:"user_id"`
	Status         string               `json:"status"`
	ModelID        string               `json:"model_id"`
	ModelVersion   string               `json:"model_version"`
	RequestedCount int                  `json:"requested_count"`
	CompletedCount int                  `json:"completed_count"`
	CreatedAt      string               `json:"created_at"`
	CompletedAt    string               `json:"completed_at,omitempty"`
	StatusURL      string               `json:"status_url"`
	Predictions    []PredictionResponse `json:"predictions,omitempty"`
}
