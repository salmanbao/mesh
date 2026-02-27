package ports

import "context"

type UserIdentity struct {
	UserID string
	Email  string
	Role   string
}

type AuthReader interface {
	GetUser(ctx context.Context, userID string) (UserIdentity, error)
}

type CampaignReader interface {
	EnsureCampaignAccessible(ctx context.Context, campaignID, userID string) error
}

type ContentLibraryReader interface {
	EnsureProductLicensed(ctx context.Context, productID, userID string) error
}

type EscrowReader interface {
	EnsureFundingSource(ctx context.Context, userID string, amount float64, currency string) error
}

type FeeEngineReader interface {
	GetFeeRate(ctx context.Context, trafficSource, tier string) (float64, error)
}

type ProductReader interface {
	EnsureProductActive(ctx context.Context, productID string) error
}
