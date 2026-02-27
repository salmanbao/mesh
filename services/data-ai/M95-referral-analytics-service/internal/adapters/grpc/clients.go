package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/ports"
)

type affiliateClient struct{ endpoint string }

func NewAffiliateClient(endpoint string) ports.AffiliateReader {
	return &affiliateClient{endpoint: endpoint}
}

func (c *affiliateClient) GetAffiliateSummary(_ context.Context, userID string, from, to time.Time) (ports.AffiliateSummary, error) {
	_ = from
	_ = to
	if strings.Contains(strings.ToLower(c.endpoint), "fail") {
		return ports.AffiliateSummary{}, errors.New("affiliate upstream unavailable")
	}
	return ports.AffiliateSummary{AffiliateID: userID, Status: "active", CommissionPct: 0.12}, nil
}
