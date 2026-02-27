package domain

import (
	"strings"
	"time"
)

type PayoutStatus string
type PayoutMethod string

const (
	PayoutStatusScheduled  PayoutStatus = "scheduled"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusPaid       PayoutStatus = "paid"
	PayoutStatusFailed     PayoutStatus = "failed"
)

const (
	PayoutMethodStandard PayoutMethod = "standard"
	PayoutMethodInstant  PayoutMethod = "instant"
)

type Payout struct {
	PayoutID      string       `json:"payout_id"`
	UserID        string       `json:"user_id"`
	SubmissionID  string       `json:"submission_id"`
	Amount        float64      `json:"amount"`
	Currency      string       `json:"currency"`
	Method        PayoutMethod `json:"method"`
	Status        PayoutStatus `json:"status"`
	FailureReason string       `json:"failure_reason,omitempty"`
	ScheduledAt   time.Time    `json:"scheduled_at"`
	ProcessingAt  *time.Time   `json:"processing_at,omitempty"`
	PaidAt        *time.Time   `json:"paid_at,omitempty"`
	FailedAt      *time.Time   `json:"failed_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

func ValidatePayoutRequestInput(userID, submissionID string, amount float64, method PayoutMethod, scheduledAt time.Time) error {
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(submissionID) == "" {
		return ErrInvalidInput
	}
	if amount <= 0 {
		return ErrInvalidInput
	}
	if method != PayoutMethodStandard && method != PayoutMethodInstant {
		return ErrInvalidInput
	}
	if scheduledAt.IsZero() {
		return ErrInvalidInput
	}
	return nil
}
