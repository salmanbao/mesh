package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
)

type ClientConfig struct {
	AuthGRPCURL         string
	CatalogGRPCURL      string
	FeeEngineGRPCURL    string
	FinanceGRPCURL      string
	SubscriptionGRPCURL string
}

type AuthClient struct{}
type CatalogClient struct{}
type FeeClient struct{}
type FinanceClient struct{}
type SubscriptionClient struct{}

func NewAuthClient(_ string) *AuthClient                 { return &AuthClient{} }
func NewCatalogClient(_ string) *CatalogClient           { return &CatalogClient{} }
func NewFeeClient(_ string) *FeeClient                   { return &FeeClient{} }
func NewFinanceClient(_ string) *FinanceClient           { return &FinanceClient{} }
func NewSubscriptionClient(_ string) *SubscriptionClient { return &SubscriptionClient{} }

func (c *AuthClient) GetUser(_ context.Context, userID string) (ports.UserIdentity, error) {
	return ports.UserIdentity{UserID: userID, Email: userID + "@example.com", Role: "user"}, nil
}

func (c *CatalogClient) GetSource(_ context.Context, sourceType, sourceID string) error {
	_ = sourceType
	_ = sourceID
	return nil
}

func (c *FeeClient) GetFeeRate(_ context.Context, sourceType string) (float64, error) {
	_ = sourceType
	return 0.0825, nil
}

func (c *FinanceClient) RecordTransaction(_ context.Context, transactionType, invoiceID string, amount float64, currency string) error {
	_ = transactionType
	_ = invoiceID
	_ = amount
	_ = currency
	return nil
}

func (c *SubscriptionClient) ValidateSubscription(_ context.Context, customerID string) error {
	_ = customerID
	return nil
}
