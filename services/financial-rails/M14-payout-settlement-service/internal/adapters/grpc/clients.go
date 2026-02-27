package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/ports"
)

type AuthClient struct{}
type ProfileClient struct{}
type BillingClient struct{}
type EscrowClient struct{}
type RiskClient struct{}
type FinanceClient struct{}
type RewardClient struct{}

func NewAuthClient(_ string) *AuthClient       { return &AuthClient{} }
func NewProfileClient(_ string) *ProfileClient { return &ProfileClient{} }
func NewBillingClient(_ string) *BillingClient { return &BillingClient{} }
func NewEscrowClient(_ string) *EscrowClient   { return &EscrowClient{} }
func NewRiskClient(_ string) *RiskClient       { return &RiskClient{} }
func NewFinanceClient(_ string) *FinanceClient { return &FinanceClient{} }
func NewRewardClient(_ string) *RewardClient   { return &RewardClient{} }

func (c *AuthClient) GetUser(_ context.Context, userID string) (ports.UserIdentity, error) {
	return ports.UserIdentity{UserID: userID, Email: userID + "@example.com", Role: "user"}, nil
}

func (c *ProfileClient) EnsurePayoutProfile(_ context.Context, userID string) error {
	_ = userID
	return nil
}

func (c *BillingClient) EnsureBillingAccount(_ context.Context, userID string) error {
	_ = userID
	return nil
}

func (c *EscrowClient) EnsureReleasable(_ context.Context, submissionID string) error {
	_ = submissionID
	return nil
}

func (c *RiskClient) EnsureEligible(_ context.Context, userID string, amount float64) error {
	_ = userID
	_ = amount
	return nil
}

func (c *FinanceClient) EnsureLiquidity(_ context.Context, userID string, amount float64, currency string) error {
	_ = userID
	_ = amount
	_ = currency
	return nil
}

func (c *RewardClient) EnsureRewardEligible(_ context.Context, submissionID, userID string, amount float64) error {
	_ = submissionID
	_ = userID
	_ = amount
	return nil
}
