package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/ports"
)

type campaignDiscoveryClient struct{ endpoint string }

func NewCampaignDiscoveryClient(endpoint string) ports.CampaignDiscoveryReader {
	return &campaignDiscoveryClient{endpoint: endpoint}
}

func (c *campaignDiscoveryClient) ListCandidateCampaigns(_ context.Context, userID, role, segment string, limit int) ([]ports.CampaignCandidate, error) {
	if strings.Contains(strings.ToLower(c.endpoint), "fail") {
		return nil, errors.New("campaign discovery upstream unavailable")
	}
	if limit <= 0 {
		limit = 10
	}
	items := make([]ports.CampaignCandidate, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, ports.CampaignCandidate{
			CampaignID:    fmt.Sprintf("cmp_%s_%02d", strings.ToLower(role), i+1),
			Title:         fmt.Sprintf("%s Campaign %02d", strings.Title(strings.ToLower(role)), i+1),
			CreatorID:     fmt.Sprintf("creator_%02d", (i%5)+1),
			Platform:      []string{"TikTok", "Instagram", "YouTube", "X"}[i%4],
			Category:      []string{"Comedy", "Gaming", "Lifestyle", "Education"}[i%4],
			RewardRate:    1.0 + float64(i%5)*0.25,
			ApprovalRate:  0.65 + float64((limit-i)%4)*0.07,
			VelocityScore: 0.4 + float64(i%6)*0.08,
			AgeDays:       (i * 11) % 120,
		})
	}
	_ = userID
	_ = segment
	return items, nil
}
