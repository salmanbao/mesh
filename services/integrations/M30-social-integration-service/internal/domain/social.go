package domain

import "time"

const (
	AccountStatusActive       = "active"
	AccountStatusRevoked      = "revoked"
	AccountStatusDisconnected = "disconnected"
)

type SocialAccount struct {
	SocialAccountID string    `json:"social_account_id"`
	UserID          string    `json:"user_id"`
	Platform        string    `json:"platform"`
	Handle          string    `json:"handle"`
	Status          string    `json:"status"`
	ConnectedAt     time.Time `json:"connected_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Source          string    `json:"source"`
}

type PostValidation struct {
	ValidationID string    `json:"validation_id"`
	UserID       string    `json:"user_id"`
	Platform     string    `json:"platform"`
	PostID       string    `json:"post_id"`
	IsValid      bool      `json:"is_valid"`
	Reason       string    `json:"reason,omitempty"`
	ValidatedAt  time.Time `json:"validated_at"`
	Source       string    `json:"source"`
}

type SocialMetric struct {
	MetricID      string    `json:"metric_id"`
	UserID        string    `json:"user_id"`
	Platform      string    `json:"platform"`
	FollowerCount int       `json:"follower_count"`
	SyncedAt      time.Time `json:"synced_at"`
}

type ComponentCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LatencyMS   int       `json:"latency_ms"`
	LastChecked time.Time `json:"last_checked"`
}

type HealthReport struct {
	Status        string                    `json:"status"`
	Timestamp     time.Time                 `json:"timestamp"`
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Version       string                    `json:"version"`
	Checks        map[string]ComponentCheck `json:"checks"`
}

func IsValidProvider(v string) bool {
	switch v {
	case "instagram", "tiktok", "youtube", "twitter", "x", "snapchat":
		return true
	default:
		return false
	}
}
