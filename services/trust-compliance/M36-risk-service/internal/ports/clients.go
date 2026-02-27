package ports

import "context"

type AuthReader interface {
	GetAuthSummary(ctx context.Context, userID string) (AuthSummary, error)
}
type ProfileReader interface {
	GetProfileSummary(ctx context.Context, userID string) (ProfileSummary, error)
}
type FraudReader interface {
	GetFraudSummary(ctx context.Context, sellerID string) (FraudSummary, error)
}
type ModerationReader interface {
	GetModerationSummary(ctx context.Context, sellerID string) (ModerationSummary, error)
}
type ResolutionReader interface {
	GetResolutionSummary(ctx context.Context, sellerID string) (ResolutionSummary, error)
}
type ReputationReader interface {
	GetReputationSummary(ctx context.Context, sellerID string) (ReputationSummary, error)
}

type AuthSummary struct {
	AccountAgeDays int
	Verified       bool
}
type ProfileSummary struct {
	SellerID         string
	AvailableBalance float64
	Country          string
}
type FraudSummary struct {
	SellerID          string
	FraudHistoryCount int
	SalesVelocity     float64
}
type ModerationSummary struct {
	SellerID            string
	ProductClarityScore float64
	RecentFlags         int
}
type ResolutionSummary struct {
	SellerID          string
	TotalDisputes12M  int
	ResolvedForSeller int
	ResolvedForBuyer  int
	PartialRefund     int
	LastDisputeDate   string
	LastOutcome       string
	DisputeRate       float64
}
type ReputationSummary struct {
	SellerID string
	Score    float64
}
