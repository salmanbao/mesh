package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
)

type AuthClient struct{}
type CampaignClient struct{}
type ContentLibraryClient struct{}
type EscrowClient struct{}
type FeeEngineClient struct{}
type ProductClient struct{}

func NewAuthClient(_ string) *AuthClient                     { return &AuthClient{} }
func NewCampaignClient(_ string) *CampaignClient             { return &CampaignClient{} }
func NewContentLibraryClient(_ string) *ContentLibraryClient { return &ContentLibraryClient{} }
func NewEscrowClient(_ string) *EscrowClient                 { return &EscrowClient{} }
func NewFeeEngineClient(_ string) *FeeEngineClient           { return &FeeEngineClient{} }
func NewProductClient(_ string) *ProductClient               { return &ProductClient{} }

func (c *AuthClient) GetUser(_ context.Context, userID string) (ports.UserIdentity, error) {
	return ports.UserIdentity{UserID: userID, Email: userID + "@example.com", Role: "user"}, nil
}

func (c *CampaignClient) EnsureCampaignAccessible(_ context.Context, campaignID, userID string) error {
	_ = campaignID
	_ = userID
	return nil
}

func (c *ContentLibraryClient) EnsureProductLicensed(_ context.Context, productID, userID string) error {
	_ = productID
	_ = userID
	return nil
}

func (c *EscrowClient) EnsureFundingSource(_ context.Context, userID string, amount float64, currency string) error {
	_ = userID
	_ = amount
	_ = currency
	return nil
}

func (c *FeeEngineClient) GetFeeRate(_ context.Context, trafficSource, tier string) (float64, error) {
	_ = trafficSource
	_ = tier
	return 0.03, nil
}

func (c *ProductClient) EnsureProductActive(_ context.Context, productID string) error {
	_ = productID
	return nil
}
