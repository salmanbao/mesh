package ports

import (
	"context"
	"time"
)

type AffiliateSummary struct {
	AffiliateID   string
	Status        string
	CommissionPct float64
}

type AffiliateReader interface {
	GetAffiliateSummary(ctx context.Context, userID string, from, to time.Time) (AffiliateSummary, error)
}
