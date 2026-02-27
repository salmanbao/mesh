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

type ProfileReader interface {
	EnsurePayoutProfile(ctx context.Context, userID string) error
}

type BillingReader interface {
	EnsureBillingAccount(ctx context.Context, userID string) error
}

type EscrowReader interface {
	EnsureReleasable(ctx context.Context, submissionID string) error
}

type RiskReader interface {
	EnsureEligible(ctx context.Context, userID string, amount float64) error
}

type FinanceReader interface {
	EnsureLiquidity(ctx context.Context, userID string, amount float64, currency string) error
}

type RewardReader interface {
	EnsureRewardEligible(ctx context.Context, submissionID, userID string, amount float64) error
}
