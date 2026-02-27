package ports

import (
	"context"
	"time"
)

type VerificationAccount struct {
	UserID      string
	Platform    string
	Handle      string
	Status      string
	ConnectedAt time.Time
}

type SocialVerificationOwnerAPI interface {
	ListUserAccounts(ctx context.Context, userID string) ([]VerificationAccount, error)
}
