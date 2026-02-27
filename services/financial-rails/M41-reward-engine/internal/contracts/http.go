package contracts

import "time"

type CalculateRewardRequest struct {
	UserID                  string    `json:"user_id"`
	SubmissionID            string    `json:"submission_id"`
	CampaignID              string    `json:"campaign_id"`
	LockedViews             int64     `json:"locked_views"`
	RatePer1K               float64   `json:"rate_per_1k"`
	FraudScore              float64   `json:"fraud_score"`
	VerificationCompletedAt time.Time `json:"verification_completed_at"`
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
