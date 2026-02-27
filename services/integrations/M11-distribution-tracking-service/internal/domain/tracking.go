package domain

import (
	"strings"
	"time"
)

type TrackedPostStatus string

const (
	TrackedPostStatusPendingAttribution TrackedPostStatus = "pending_attribution"
	TrackedPostStatusActive             TrackedPostStatus = "active"
	TrackedPostStatusArchived           TrackedPostStatus = "archived"
)

type TrackedPost struct {
	TrackedPostID      string            `json:"tracked_post_id"`
	UserID             string            `json:"user_id"`
	Platform           string            `json:"platform"`
	PostURL            string            `json:"post_url"`
	DistributionItemID string            `json:"distribution_item_id,omitempty"`
	CampaignID         string            `json:"campaign_id,omitempty"`
	Status             TrackedPostStatus `json:"status"`
	ValidationStatus   string            `json:"validation_status"`
	LastPolledAt       *time.Time        `json:"last_polled_at,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type MetricSnapshot struct {
	SnapshotID    string    `json:"snapshot_id"`
	TrackedPostID string    `json:"tracked_post_id"`
	Platform      string    `json:"platform"`
	Views         int       `json:"views"`
	Likes         int       `json:"likes"`
	Shares        int       `json:"shares"`
	Comments      int       `json:"comments"`
	PolledAt      time.Time `json:"polled_at"`
}

func IsValidPlatform(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "tiktok", "instagram", "youtube", "x", "twitter", "facebook":
		return true
	default:
		return false
	}
}
