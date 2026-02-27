package domain

import (
	"math"
	"strings"
	"time"
)

type RewardStatus string

const (
	RewardStatusCalculated     RewardStatus = "calculated"
	RewardStatusBelowThreshold RewardStatus = "below_threshold"
	RewardStatusEligible       RewardStatus = "reward_eligible"
	RewardStatusFraudRejected  RewardStatus = "fraud_rejected"
	RewardStatusCancelled      RewardStatus = "cancelled"
)

type Reward struct {
	SubmissionID            string       `json:"submission_id"`
	UserID                  string       `json:"user_id"`
	CampaignID              string       `json:"campaign_id"`
	LockedViews             int64        `json:"locked_views"`
	RatePer1K               float64      `json:"rate_per_1k"`
	GrossAmount             float64      `json:"gross_amount"`
	NetAmount               float64      `json:"net_amount"`
	RolloverApplied         float64      `json:"rollover_applied"`
	RolloverBalance         float64      `json:"rollover_balance"`
	FraudScore              float64      `json:"fraud_score"`
	Status                  RewardStatus `json:"status"`
	VerificationCompletedAt time.Time    `json:"verification_completed_at"`
	CalculatedAt            time.Time    `json:"calculated_at"`
	EligibleAt              *time.Time   `json:"eligible_at,omitempty"`
	LastEventID             string       `json:"last_event_id,omitempty"`
}

type RolloverBalance struct {
	UserID    string    `json:"user_id"`
	Balance   float64   `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ValidateCalculationInput(userID, submissionID, campaignID string, lockedViews int64, ratePer1K float64) error {
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(submissionID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(campaignID) == "" {
		return ErrInvalidInput
	}
	if lockedViews < 0 {
		return ErrInvalidInput
	}
	if ratePer1K < 0 {
		return ErrInvalidInput
	}
	return nil
}

func CalculateGrossAmount(lockedViews int64, ratePer1K float64) float64 {
	if lockedViews < 0 || ratePer1K < 0 {
		return 0
	}
	amount := (float64(lockedViews) / 1000.0) * ratePer1K
	return RoundCurrency(amount, 4)
}

func RoundCurrency(value float64, places int) float64 {
	if places < 0 {
		places = 0
	}
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
}
