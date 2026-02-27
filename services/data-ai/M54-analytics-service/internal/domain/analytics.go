package domain

import (
	"strings"
	"time"
)

type FactSubmission struct {
	SubmissionID string    `json:"submission_id"`
	CreatorID    string    `json:"creator_id"`
	CampaignID   string    `json:"campaign_id"`
	Platform     string    `json:"platform"`
	Status       string    `json:"status"`
	Views        int64     `json:"views"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type FactPayout struct {
	PayoutID    string    `json:"payout_id"`
	CreatorID   string    `json:"creator_id"`
	Amount      float64   `json:"amount"`
	OccurredAt  time.Time `json:"occurred_at"`
	SourceEvent string    `json:"source_event"`
}

type FactTransaction struct {
	TransactionID string    `json:"transaction_id"`
	UserID        string    `json:"user_id"`
	Amount        float64   `json:"amount"`
	Refunded      bool      `json:"refunded"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type FactClick struct {
	ClickID     string    `json:"click_id"`
	UserID      string    `json:"user_id"`
	Platform    string    `json:"platform"`
	ItemType    string    `json:"item_type"`
	SessionID   string    `json:"session_id"`
	OccurredAt  time.Time `json:"occurred_at"`
	SourceEvent string    `json:"source_event"`
}

type DimUser struct {
	UserID           string    `json:"user_id"`
	Role             string    `json:"role"`
	Country          string    `json:"country"`
	ConsentAnalytics bool      `json:"consent_analytics"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type DimCampaign struct {
	CampaignID string    `json:"campaign_id"`
	BrandID    string    `json:"brand_id"`
	Category   string    `json:"category"`
	RewardRate float64   `json:"reward_rate"`
	Budget     float64   `json:"budget"`
	LaunchedAt time.Time `json:"launched_at"`
}

type DailyEarnings struct {
	DayDate       string    `json:"day_date"`
	CreatorID     string    `json:"creator_id"`
	GrossEarnings float64   `json:"gross_earnings"`
	NetEarnings   float64   `json:"net_earnings"`
	Payouts       float64   `json:"payouts"`
	Refunds       float64   `json:"refunds"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreatorDashboard struct {
	UserID          string             `json:"user_id"`
	DateFrom        string             `json:"date_from"`
	DateTo          string             `json:"date_to"`
	GeneratedAt     time.Time          `json:"generated_at"`
	Submissions     int                `json:"submissions"`
	Approved        int                `json:"approved"`
	TotalViews      int64              `json:"total_views"`
	TotalEarnings   float64            `json:"total_earnings"`
	TotalPayouts    float64            `json:"total_payouts"`
	TotalRefunds    float64            `json:"total_refunds"`
	TopPlatforms    map[string]int     `json:"top_platforms"`
	DataFreshnessS  int                `json:"data_freshness_seconds"`
	SourceBreakdown map[string]float64 `json:"source_breakdown"`
}

type TopCreator struct {
	UserID      string  `json:"user_id"`
	Earnings    float64 `json:"earnings"`
	Submissions int     `json:"submissions"`
	RefundRate  float64 `json:"refund_rate"`
}

type FinancialReport struct {
	DateFrom             string       `json:"date_from"`
	DateTo               string       `json:"date_to"`
	GeneratedAt          time.Time    `json:"generated_at"`
	GMV                  float64      `json:"gmv"`
	NetRevenue           float64      `json:"net_revenue"`
	TotalPayoutLiability float64      `json:"total_payout_liability"`
	RefundRate           float64      `json:"refund_rate"`
	TopCreators          []TopCreator `json:"top_creators"`
}

type ExportJobStatus string

const (
	ExportStatusQueued ExportJobStatus = "queued"
	ExportStatusReady  ExportJobStatus = "ready"
)

type ExportJob struct {
	ExportID       string            `json:"export_id"`
	UserID         string            `json:"user_id"`
	ReportType     string            `json:"report_type"`
	Format         string            `json:"format"`
	DateFrom       string            `json:"date_from"`
	DateTo         string            `json:"date_to"`
	Filters        map[string]string `json:"filters"`
	Status         ExportJobStatus   `json:"status"`
	DownloadURL    string            `json:"download_url"`
	IdempotencyKey string            `json:"idempotency_key"`
	CreatedAt      time.Time         `json:"created_at"`
	ReadyAt        *time.Time        `json:"ready_at,omitempty"`
}

func ValidateExportFormat(format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv", "json":
		return nil
	default:
		return ErrInvalidInput
	}
}

func ValidateReportType(reportType string) error {
	switch strings.ToLower(strings.TrimSpace(reportType)) {
	case "creator_dashboard", "admin_financial_report":
		return nil
	default:
		return ErrInvalidInput
	}
}

func NormalizeReportType(reportType string) string {
	value := strings.ToLower(strings.TrimSpace(reportType))
	if value == "" {
		return "creator_dashboard"
	}
	return value
}
