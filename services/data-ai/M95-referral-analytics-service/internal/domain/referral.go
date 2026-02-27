package domain

import (
	"strings"
	"time"
)

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"

	ExportStatusQueued     = "queued"
	ExportStatusProcessing = "processing"
	ExportStatusCompleted  = "completed"
	ExportStatusFailed     = "failed"
)

type ReferralEvent struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	ReferralToken string    `json:"referral_token,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	ConversionAmt float64   `json:"conversion_amount,omitempty"`
	Platform      string    `json:"platform,omitempty"`
	UTMSource     string    `json:"utm_source,omitempty"`
	UTMMedium     string    `json:"utm_medium,omitempty"`
	UTMCampaign   string    `json:"utm_campaign,omitempty"`
	Country       string    `json:"country,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type ReferralAggregateDaily struct {
	ID            string    `json:"id"`
	Date          string    `json:"date"`
	ReferralToken string    `json:"referral_token,omitempty"`
	Platform      string    `json:"platform,omitempty"`
	Clicks        int       `json:"clicks"`
	Signups       int       `json:"signups"`
	Conversions   int       `json:"conversions"`
	Revenue       float64   `json:"revenue"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ReferralFunnelAggregate struct {
	ID              string    `json:"id"`
	Date            string    `json:"date"`
	Clicks          int       `json:"clicks"`
	Signups         int       `json:"signups"`
	FirstPurchases  int       `json:"first_purchases"`
	RepeatPurchases int       `json:"repeat_purchases"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ReferralCohortRetention struct {
	ID                 string    `json:"id"`
	CohortDate         string    `json:"cohort_date"`
	CohortSize         int       `json:"cohort_size"`
	Day7Rate           float64   `json:"day7_rate"`
	Day30Rate          float64   `json:"day30_rate"`
	Day90Rate          float64   `json:"day90_rate"`
	RepeatPurchaseRate float64   `json:"repeat_purchase_rate"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ReferralGeoAggregate struct {
	ID          string    `json:"id"`
	Date        string    `json:"date"`
	Country     string    `json:"country"`
	Clicks      int       `json:"clicks"`
	Conversions int       `json:"conversions"`
	Revenue     float64   `json:"revenue"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReferralExportJob struct {
	ID             string            `json:"id"`
	RequestedBy    string            `json:"requested_by"`
	ExportType     string            `json:"export_type"`
	Format         string            `json:"format"`
	Period         string            `json:"period"`
	Filters        map[string]string `json:"filters,omitempty"`
	Status         string            `json:"status"`
	OutputURI      string            `json:"output_uri,omitempty"`
	IdempotencyKey string            `json:"-"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
}

type FunnelReport struct {
	StartDate         string    `json:"start_date"`
	EndDate           string    `json:"end_date"`
	Clicks            int       `json:"clicks"`
	Signups           int       `json:"signups"`
	FirstPurchases    int       `json:"first_purchases"`
	RepeatPurchases   int       `json:"repeat_purchases"`
	ClickToSignupRate float64   `json:"click_to_signup_rate"`
	SignupToFirstRate float64   `json:"signup_to_first_purchase_rate"`
	FirstToRepeatRate float64   `json:"first_to_repeat_purchase_rate"`
	DataFreshnessAt   time.Time `json:"data_freshness_at"`
}

type LeaderboardEntry struct {
	Rank           int     `json:"rank"`
	ReferralToken  string  `json:"referral_token"`
	Sales          float64 `json:"sales"`
	Conversions    int     `json:"conversions"`
	Clicks         int     `json:"clicks"`
	ConversionRate float64 `json:"conversion_rate"`
	LTV90          float64 `json:"ltv_90"`
}

type LeaderboardReport struct {
	Period        string             `json:"period"`
	TopPerformers []LeaderboardEntry `json:"top_performers"`
	GeneratedAt   time.Time          `json:"generated_at"`
}

type CohortRetentionReport struct {
	CohortStart string                    `json:"cohort_start"`
	CohortEnd   string                    `json:"cohort_end"`
	Cohorts     []ReferralCohortRetention `json:"cohorts"`
	GeneratedAt time.Time                 `json:"generated_at"`
}

type GeoPerformanceReport struct {
	StartDate    string                 `json:"start_date"`
	EndDate      string                 `json:"end_date"`
	TopCountries []ReferralGeoAggregate `json:"top_countries"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

type PayoutForecast struct {
	Period           string    `json:"period"`
	ForecastedAmount float64   `json:"forecasted_amount"`
	ConfidenceLow    float64   `json:"confidence_low"`
	ConfidenceHigh   float64   `json:"confidence_high"`
	DeviationAlert   bool      `json:"deviation_alert"`
	GeneratedAt      time.Time `json:"generated_at"`
}

func NormalizeExportType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "leaderboard":
		return "leaderboard"
	case "funnel":
		return "funnel"
	case "cohorts", "cohorts_retention", "cohort_retention":
		return "cohorts"
	case "geo":
		return "geo"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func ValidateExportType(raw string) error {
	switch NormalizeExportType(raw) {
	case "leaderboard", "funnel", "cohorts", "geo":
		return nil
	default:
		return ErrInvalidInput
	}
}

func NormalizeExportFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "csv":
		return "csv"
	case "json":
		return "json"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func ValidateExportFormat(raw string) error {
	switch NormalizeExportFormat(raw) {
	case "csv", "json":
		return nil
	default:
		return ErrInvalidInput
	}
}

func NormalizePeriod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "30d":
		return "30d"
	case "7d", "90d", "all":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func ValidatePeriod(raw string) error {
	switch NormalizePeriod(raw) {
	case "7d", "30d", "90d", "all":
		return nil
	default:
		return ErrInvalidInput
	}
}

func SafeRate(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / float64(d)
}
