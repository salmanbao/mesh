package ports

import "context"

type ProfileSnapshot struct {
	UserID string
	Role   string
}

type BillingSnapshot struct {
	PendingBalance   float64
	AvailableBalance float64
}

type CampaignSnapshot struct {
	CampaignID   string
	Name         string
	Submissions  int
	AverageViews int64
}

type OnboardingSnapshot struct {
	IsComplete bool
}

type FinanceSnapshot struct {
	PendingPayouts float64
	LastPayout     float64
}

type EscrowSnapshot struct {
	LockedAmount float64
}

type RewardSnapshot struct {
	TotalRewards float64
}

type GamificationSnapshot struct {
	Badges []string
}

type AnalyticsSnapshot struct {
	TotalEarnings float64
	TotalViews    int64
	Submissions   int
}

type ProductSnapshot struct {
	PublishedApps int
	AppRevenue    float64
}

type ProfileReader interface {
	GetProfile(ctx context.Context, userID string) (ProfileSnapshot, error)
}

type BillingReader interface {
	GetBillingSummary(ctx context.Context, userID string) (BillingSnapshot, error)
}

type ContentReader interface {
	ListCampaigns(ctx context.Context, userID string) ([]CampaignSnapshot, error)
}

type EscrowReader interface {
	GetEscrowSummary(ctx context.Context, userID string) (EscrowSnapshot, error)
}

type OnboardingReader interface {
	GetOnboarding(ctx context.Context, userID string) (OnboardingSnapshot, error)
}

type FinanceReader interface {
	GetFinanceSummary(ctx context.Context, userID string) (FinanceSnapshot, error)
}

type RewardReader interface {
	GetRewardSummary(ctx context.Context, userID string) (RewardSnapshot, error)
}

type GamificationReader interface {
	GetGamification(ctx context.Context, userID string) (GamificationSnapshot, error)
}

type AnalyticsReader interface {
	GetDashboardMetrics(ctx context.Context, userID, role, dateRange string) (AnalyticsSnapshot, error)
}

type ProductReader interface {
	GetProductSummary(ctx context.Context, userID string) (ProductSnapshot, error)
}
