package ports

import (
	"context"
	"time"
)

type CampaignDiscoveryReader interface {
	ListCandidateCampaigns(ctx context.Context, userID, role, segment string, limit int) ([]CampaignCandidate, error)
}

type CampaignCandidate struct {
	CampaignID    string
	Title         string
	CreatorID     string
	Platform      string
	Category      string
	RewardRate    float64
	ApprovalRate  float64
	VelocityScore float64
	AgeDays       int
}

type DiscoverySummary struct {
	Count      int
	FetchedAt  time.Time
	SourceMode string
}
