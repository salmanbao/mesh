package domain

import (
	"strings"
	"time"
)

const (
	LicenseDecisionAllowed = "allowed"
	LicenseDecisionHeld    = "license_hold_applied"
)

const (
	LicenseHoldStatusPendingReview = "pending_license_review"
	LicenseAppealStatusPending     = "pending_review"
	DMCATakedownStatusReceived     = "takedown_notice_received"
)

type CopyrightMatch struct {
	MatchID          string    `json:"match_id"`
	SubmissionID     string    `json:"submission_id"`
	CreatorID        string    `json:"creator_id"`
	MediaType        string    `json:"media_type"`
	MediaURL         string    `json:"media_url"`
	ConfidenceScore  float64   `json:"confidence_score"`
	MatchedTitle     string    `json:"matched_title"`
	RightsHolderName string    `json:"rights_holder_name"`
	CreatedAt        time.Time `json:"created_at"`
}

type LicenseHold struct {
	HoldID        string    `json:"hold_id"`
	SubmissionID  string    `json:"submission_id"`
	MatchID       string    `json:"match_id"`
	CreatorID     string    `json:"creator_id"`
	Reason        string    `json:"reason"`
	Status        string    `json:"status"`
	HoldCreatedAt time.Time `json:"hold_created_at"`
}

type LicenseAppeal struct {
	AppealID           string    `json:"appeal_id"`
	SubmissionID       string    `json:"submission_id"`
	HoldID             string    `json:"hold_id"`
	CreatorID          string    `json:"creator_id"`
	CreatorExplanation string    `json:"creator_explanation"`
	Status             string    `json:"status"`
	AppealCreatedAt    time.Time `json:"appeal_created_at"`
}

type DMCATakedown struct {
	DMCAID           string    `json:"dmca_id"`
	SubmissionID     string    `json:"submission_id"`
	RightsHolder     string    `json:"rights_holder_name"`
	ContactEmail     string    `json:"contact_email"`
	Reference        string    `json:"reference"`
	Status           string    `json:"status"`
	NoticeReceivedAt time.Time `json:"notice_received_at"`
}

type AuditLog struct {
	EventID      string            `json:"event_id"`
	EventType    string            `json:"event_type"`
	EntityID     string            `json:"entity_id"`
	SubmissionID string            `json:"submission_id"`
	ActorID      string            `json:"actor_id"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

func IsValidMediaType(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "audio", "video":
		return true
	default:
		return false
	}
}
