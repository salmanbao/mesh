package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/ports"
)

func endpointFailing(endpoint string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(endpoint)), "fail")
}

type authClient struct{ endpoint string }

type profileClient struct{ endpoint string }

type fraudClient struct{ endpoint string }

type moderationClient struct{ endpoint string }

type resolutionClient struct{ endpoint string }

type reputationClient struct{ endpoint string }

func NewAuthClient(endpoint string) ports.AuthReader       { return &authClient{endpoint: endpoint} }
func NewProfileClient(endpoint string) ports.ProfileReader { return &profileClient{endpoint: endpoint} }
func NewFraudClient(endpoint string) ports.FraudReader     { return &fraudClient{endpoint: endpoint} }
func NewModerationClient(endpoint string) ports.ModerationReader {
	return &moderationClient{endpoint: endpoint}
}
func NewResolutionClient(endpoint string) ports.ResolutionReader {
	return &resolutionClient{endpoint: endpoint}
}
func NewReputationClient(endpoint string) ports.ReputationReader {
	return &reputationClient{endpoint: endpoint}
}

func (c *authClient) GetAuthSummary(_ context.Context, userID string) (ports.AuthSummary, error) {
	if endpointFailing(c.endpoint) {
		return ports.AuthSummary{}, errors.New("auth upstream unavailable")
	}
	age := 120
	if strings.Contains(userID, "new") {
		age = 5
	}
	return ports.AuthSummary{AccountAgeDays: age, Verified: true}, nil
}

func (c *profileClient) GetProfileSummary(_ context.Context, userID string) (ports.ProfileSummary, error) {
	if endpointFailing(c.endpoint) {
		return ports.ProfileSummary{}, errors.New("profile upstream unavailable")
	}
	return ports.ProfileSummary{SellerID: userID, AvailableBalance: 1250.0, Country: "US"}, nil
}

func (c *fraudClient) GetFraudSummary(_ context.Context, sellerID string) (ports.FraudSummary, error) {
	if endpointFailing(c.endpoint) {
		return ports.FraudSummary{}, errors.New("fraud upstream unavailable")
	}
	return ports.FraudSummary{SellerID: sellerID, FraudHistoryCount: 1, SalesVelocity: 0.42}, nil
}

func (c *moderationClient) GetModerationSummary(_ context.Context, sellerID string) (ports.ModerationSummary, error) {
	if endpointFailing(c.endpoint) {
		return ports.ModerationSummary{}, errors.New("moderation upstream unavailable")
	}
	return ports.ModerationSummary{SellerID: sellerID, ProductClarityScore: 0.88, RecentFlags: 0}, nil
}

func (c *resolutionClient) GetResolutionSummary(_ context.Context, sellerID string) (ports.ResolutionSummary, error) {
	if endpointFailing(c.endpoint) {
		return ports.ResolutionSummary{}, errors.New("resolution upstream unavailable")
	}
	return ports.ResolutionSummary{SellerID: sellerID, TotalDisputes12M: 2, ResolvedForSeller: 1, ResolvedForBuyer: 1, PartialRefund: 0, LastDisputeDate: "2026-01-15", LastOutcome: "resolved_for_seller", DisputeRate: 0.012}, nil
}

func (c *reputationClient) GetReputationSummary(_ context.Context, sellerID string) (ports.ReputationSummary, error) {
	if endpointFailing(c.endpoint) {
		return ports.ReputationSummary{}, errors.New("reputation upstream unavailable")
	}
	return ports.ReputationSummary{SellerID: sellerID, Score: 4.7}, nil
}
