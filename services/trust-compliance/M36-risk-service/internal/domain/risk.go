package domain

import (
	"strings"
	"time"
)

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

type ReleaseScheduleItem struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type ReserveStatus struct {
	PercentageHeld           int                   `json:"percentage_held"`
	EscrowedAmount           float64               `json:"escrowed_amount"`
	AvailableBalance         float64               `json:"available_balance"`
	NextReleaseMilestone     string                `json:"next_release_milestone"`
	EstimatedReleaseSchedule []ReleaseScheduleItem `json:"estimated_release_schedule"`
}

type DisputeHistoryBreakdown struct {
	ResolvedForSeller int `json:"resolved_for_seller"`
	ResolvedForBuyer  int `json:"resolved_for_buyer"`
	PartialRefund     int `json:"partial_refund"`
}

type DisputeHistory struct {
	TotalDisputes12M   int                     `json:"total_disputes_12m"`
	Breakdown          DisputeHistoryBreakdown `json:"breakdown"`
	LastDisputeDate    string                  `json:"last_dispute_date,omitempty"`
	LastDisputeOutcome string                  `json:"last_dispute_outcome,omitempty"`
}

type SellerFlag struct {
	Reason      string `json:"reason"`
	Date        string `json:"date"`
	ActionTaken string `json:"action_taken"`
	Status      string `json:"status"`
}

type FraudAlert struct {
	FlagID            string  `json:"flag_id"`
	PatternType       string  `json:"pattern_type"`
	ConfidenceScore   float64 `json:"confidence_score"`
	RecommendedAction string  `json:"recommended_action"`
	CreatedAt         string  `json:"created_at"`
}

type RiskDashboard struct {
	CurrentRiskScore float64        `json:"current_risk_score"`
	RiskLevel        string         `json:"risk_level"`
	ReserveStatus    ReserveStatus  `json:"reserve_status"`
	DisputeHistory   DisputeHistory `json:"dispute_history"`
	RecentFlags      []SellerFlag   `json:"recent_flags"`
	FraudAlerts      []FraudAlert   `json:"fraud_alerts"`
	Alerts           []string       `json:"alerts"`
}

type SellerRiskProfile struct {
	SellerID            string    `json:"seller_id"`
	CurrentRiskScore    float64   `json:"current_risk_score"`
	PreviousRiskScore   float64   `json:"previous_risk_score"`
	RiskLevel           string    `json:"risk_level"`
	DisputeRate         float64   `json:"dispute_rate"`
	AccountAgeDays      int       `json:"account_age_days"`
	SalesVelocity       float64   `json:"sales_velocity"`
	ProductClarityScore float64   `json:"product_clarity_score"`
	FraudHistoryCount   int       `json:"fraud_history_count"`
	ReservePercentage   int       `json:"reserve_percentage"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SellerEscrow struct {
	EscrowID             string     `json:"escrow_id"`
	SellerID             string     `json:"seller_id"`
	EscrowedAmount       float64    `json:"escrowed_amount"`
	AvailableBalance     float64    `json:"available_balance"`
	ReservePercentage    int        `json:"reserve_percentage"`
	NextReleaseMilestone *time.Time `json:"next_release_milestone,omitempty"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type DisputeLog struct {
	DisputeID              string     `json:"dispute_id"`
	TransactionID          string     `json:"transaction_id"`
	SellerID               string     `json:"seller_id"`
	BuyerID                string     `json:"buyer_id"`
	DisputeType            string     `json:"dispute_type"`
	Reason                 string     `json:"reason"`
	BuyerClaim             string     `json:"buyer_claim"`
	Status                 string     `json:"status"`
	SellerResponseDeadline *time.Time `json:"seller_response_deadline,omitempty"`
	FiledAt                time.Time  `json:"filed_at"`
	ResolvedAt             *time.Time `json:"resolved_at,omitempty"`
	ResolutionStatus       string     `json:"resolution_status,omitempty"`
	RefundAmount           float64    `json:"refund_amount,omitempty"`
	EvidenceCount          int        `json:"evidence_count"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type DisputeEvidence struct {
	EvidenceID  string    `json:"evidence_id"`
	DisputeID   string    `json:"dispute_id"`
	SellerID    string    `json:"seller_id"`
	Filename    string    `json:"filename"`
	Description string    `json:"description"`
	FileURL     string    `json:"file_url"`
	SizeBytes   int64     `json:"size_bytes"`
	MimeType    string    `json:"mime_type"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

type FraudPatternFlag struct {
	FlagID            string    `json:"flag_id"`
	SellerID          string    `json:"seller_id"`
	PatternType       string    `json:"pattern_type"`
	ConfidenceScore   float64   `json:"confidence_score"`
	RecommendedAction string    `json:"recommended_action"`
	CreatedAt         time.Time `json:"created_at"`
}

type ReserveTriggerLog struct {
	TriggerID           string    `json:"trigger_id"`
	SellerID            string    `json:"seller_id"`
	TriggerType         string    `json:"trigger_type"`
	Reason              string    `json:"reason"`
	AppliedReservePct   int       `json:"applied_reserve_pct"`
	ReserveChangeAmount float64   `json:"reserve_change_amount"`
	CreatedAt           time.Time `json:"created_at"`
}

type SellerDebtLog struct {
	DebtID          string    `json:"debt_id"`
	SellerID        string    `json:"seller_id"`
	Reason          string    `json:"reason"`
	ChargeID        string    `json:"charge_id,omitempty"`
	Amount          float64   `json:"amount"`
	FeeAmount       float64   `json:"fee_amount"`
	TotalDeducted   float64   `json:"total_deducted"`
	DeductedFrom    string    `json:"deducted_from"`
	NegativeBalance bool      `json:"negative_balance"`
	CreatedAt       time.Time `json:"created_at"`
}

type SellerSuspensionLog struct {
	SuspensionID         string     `json:"suspension_id"`
	SellerID             string     `json:"seller_id"`
	Reason               string     `json:"reason"`
	SuspensionTrigger    string     `json:"suspension_trigger"`
	TargetResolutionDate *time.Time `json:"target_resolution_date,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

type ChargebackWebhook struct {
	EventID          string
	EventType        string
	OccurredAt       time.Time
	SourceService    string
	TraceID          string
	SchemaVersion    string
	PartitionKeyPath string
	PartitionKey     string
	Amount           float64
	ChargeID         string
	Currency         string
	DisputeReason    string
	SellerID         string
}

func NormalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "seller":
		return "seller"
	case "admin":
		return "admin"
	case "finance":
		return "finance"
	case "moderator":
		return "moderator"
	default:
		return ""
	}
}

func NormalizeDisputeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "refund_request":
		return "refund_request"
	case "chargeback":
		return "chargeback"
	default:
		return "refund_request"
	}
}

func RiskLevel(score float64) string {
	switch {
	case score >= 0.85:
		return "Critical"
	case score >= 0.60:
		return "High"
	case score >= 0.30:
		return "Medium"
	default:
		return "Low"
	}
}

func ReservePercentageForScore(score float64) int {
	switch {
	case score >= 0.85:
		return 100
	case score >= 0.60:
		return 75
	case score >= 0.30:
		return 35
	default:
		return 10
	}
}

func ClampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func ScoreFromSignals(disputeRate float64, accountAgeDays int, salesVelocity float64, clarityScore float64, fraudHistoryCount int) float64 {
	ageRisk := 0.1
	switch {
	case accountAgeDays < 7:
		ageRisk = 0.9
	case accountAgeDays < 30:
		ageRisk = 0.5
	case accountAgeDays < 180:
		ageRisk = 0.3
	}
	fraudRisk := float64(fraudHistoryCount) * 0.2
	if fraudRisk > 1 {
		fraudRisk = 1
	}
	clarityRisk := ClampScore(1 - clarityScore)
	score := disputeRate*0.25 + ageRisk*0.20 + ClampScore(salesVelocity)*0.25 + clarityRisk*0.15 + fraudRisk*0.15
	return ClampScore(score)
}
