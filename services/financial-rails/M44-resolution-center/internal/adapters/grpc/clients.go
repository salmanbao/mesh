package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/ports"
)

type moderationClient struct{ endpoint string }

func NewModerationClient(endpoint string) ports.ModerationReader {
	return &moderationClient{endpoint: endpoint}
}

func (c *moderationClient) GetModerationSummary(_ context.Context, userID string) (ports.ModerationSummary, error) {
	if strings.Contains(strings.ToLower(c.endpoint), "fail") {
		return ports.ModerationSummary{}, errors.New("moderation upstream unavailable")
	}
	return ports.ModerationSummary{UserID: userID, TrustScore: 0.92, RecentFlags: 0, LastReviewedAt: time.Now().UTC().Add(-24 * time.Hour)}, nil
}
