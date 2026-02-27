package ports

import (
	"context"
	"time"
)

type ModerationReader interface {
	GetModerationSummary(ctx context.Context, userID string) (ModerationSummary, error)
}

type ModerationSummary struct {
	UserID         string
	TrustScore     float64
	RecentFlags    int
	LastReviewedAt time.Time
}
