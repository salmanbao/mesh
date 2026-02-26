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

type CatalogReader interface {
	GetSource(ctx context.Context, sourceType, sourceID string) error
}

type FeeReader interface {
	GetFeeRate(ctx context.Context, sourceType string) (float64, error)
}

type FinanceWriter interface {
	RecordTransaction(ctx context.Context, transactionType, invoiceID string, amount float64, currency string) error
}

type SubscriptionReader interface {
	ValidateSubscription(ctx context.Context, customerID string) error
}
