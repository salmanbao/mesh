package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/ports"
)

type profileClient struct{ endpoint string }

type billingClient struct{ endpoint string }

type contentClient struct{ endpoint string }

type escrowClient struct{ endpoint string }

type onboardingClient struct{ endpoint string }

type financeClient struct{ endpoint string }

type rewardClient struct{ endpoint string }

type gamificationClient struct{ endpoint string }

type analyticsClient struct{ endpoint string }

type productClient struct{ endpoint string }

func NewProfileClient(endpoint string) ports.ProfileReader { return &profileClient{endpoint: endpoint} }
func NewBillingClient(endpoint string) ports.BillingReader { return &billingClient{endpoint: endpoint} }
func NewContentClient(endpoint string) ports.ContentReader { return &contentClient{endpoint: endpoint} }
func NewEscrowClient(endpoint string) ports.EscrowReader   { return &escrowClient{endpoint: endpoint} }
func NewOnboardingClient(endpoint string) ports.OnboardingReader {
	return &onboardingClient{endpoint: endpoint}
}
func NewFinanceClient(endpoint string) ports.FinanceReader { return &financeClient{endpoint: endpoint} }
func NewRewardClient(endpoint string) ports.RewardReader   { return &rewardClient{endpoint: endpoint} }
func NewGamificationClient(endpoint string) ports.GamificationReader {
	return &gamificationClient{endpoint: endpoint}
}
func NewAnalyticsClient(endpoint string) ports.AnalyticsReader {
	return &analyticsClient{endpoint: endpoint}
}
func NewProductClient(endpoint string) ports.ProductReader { return &productClient{endpoint: endpoint} }

func failForEndpoint(endpoint string) error {
	if strings.Contains(strings.ToLower(endpoint), "fail") {
		return errors.New("upstream unavailable")
	}
	return nil
}

func (c *profileClient) GetProfile(_ context.Context, userID string) (ports.ProfileSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.ProfileSnapshot{}, err
	}
	return ports.ProfileSnapshot{UserID: userID, Role: "creator"}, nil
}

func (c *billingClient) GetBillingSummary(_ context.Context, _ string) (ports.BillingSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.BillingSnapshot{}, err
	}
	return ports.BillingSnapshot{PendingBalance: 14.75, AvailableBalance: 120.10}, nil
}

func (c *contentClient) ListCampaigns(_ context.Context, _ string) ([]ports.CampaignSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return nil, err
	}
	return []ports.CampaignSnapshot{{CampaignID: "camp-001", Name: "Spring Launch", Submissions: 12, AverageViews: 8400}}, nil
}

func (c *escrowClient) GetEscrowSummary(_ context.Context, _ string) (ports.EscrowSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.EscrowSnapshot{}, err
	}
	return ports.EscrowSnapshot{LockedAmount: 45.00}, nil
}

func (c *onboardingClient) GetOnboarding(_ context.Context, _ string) (ports.OnboardingSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.OnboardingSnapshot{}, err
	}
	return ports.OnboardingSnapshot{IsComplete: true}, nil
}

func (c *financeClient) GetFinanceSummary(_ context.Context, _ string) (ports.FinanceSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.FinanceSnapshot{}, err
	}
	return ports.FinanceSnapshot{PendingPayouts: 22.50, LastPayout: 80.00}, nil
}

func (c *rewardClient) GetRewardSummary(_ context.Context, _ string) (ports.RewardSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.RewardSnapshot{}, err
	}
	return ports.RewardSnapshot{TotalRewards: 255.40}, nil
}

func (c *gamificationClient) GetGamification(_ context.Context, _ string) (ports.GamificationSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.GamificationSnapshot{}, err
	}
	return ports.GamificationSnapshot{Badges: []string{"starter", "top_creator"}}, nil
}

func (c *analyticsClient) GetDashboardMetrics(_ context.Context, _ string, _ string, _ string) (ports.AnalyticsSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.AnalyticsSnapshot{}, err
	}
	return ports.AnalyticsSnapshot{TotalEarnings: 520.30, TotalViews: 102340, Submissions: 34}, nil
}

func (c *productClient) GetProductSummary(_ context.Context, _ string) (ports.ProductSnapshot, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.ProductSnapshot{}, err
	}
	return ports.ProductSnapshot{PublishedApps: 2, AppRevenue: 1400.0}, nil
}
