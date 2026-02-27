package contracts

import "time"

type RequestPayoutRequest struct {
	UserID       string    `json:"user_id"`
	SubmissionID string    `json:"submission_id"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	Method       string    `json:"method"`
	ScheduledAt  time.Time `json:"scheduled_at"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
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
