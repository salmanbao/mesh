package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/ports"
)

type affiliateClient struct{ endpoint string }

func NewAffiliateClient(endpoint string) ports.AffiliateReader {
	return &affiliateClient{endpoint: endpoint}
}
func (c *affiliateClient) GetAffiliateSummary(_ context.Context, affiliateID string) (ports.AffiliateSummary, error) {
	if strings.Contains(strings.ToLower(c.endpoint), "fail") {
		return ports.AffiliateSummary{}, errors.New("affiliate upstream unavailable")
	}
	return ports.AffiliateSummary{AffiliateID: affiliateID, Status: "active", Region: "US"}, nil
}
