package ports

import "context"

type AffiliateSummary struct {
	AffiliateID  string
	Status       string
	Region       string
	RiskOverride *float64
}

type AffiliateReader interface {
	GetAffiliateSummary(ctx context.Context, affiliateID string) (AffiliateSummary, error)
}
