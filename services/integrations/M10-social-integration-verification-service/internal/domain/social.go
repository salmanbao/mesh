package domain

import "time"

const (
	AccountStatusPending = "pending"
	AccountStatusActive  = "active"
	AccountStatusExpired = "expired"
	AccountStatusRevoked = "revoked"
)

type SocialAccount struct {
	SocialAccountID  string
	UserID           string
	Provider         string
	Handle           string
	Status           string
	AccessToken      string
	RefreshToken     string
	TokenExpiresAt   *time.Time
	ConnectedAt      time.Time
	DisconnectedAt   *time.Time
	UpdatedAt        time.Time
}

type SocialMetric struct {
	MetricID         string
	SocialAccountID  string
	UserID           string
	Provider         string
	FollowerCount    int
	SyncedAt         time.Time
}

func IsValidProvider(v string) bool {
	switch v {
	case "instagram", "tiktok", "youtube", "twitter", "x":
		return true
	default:
		return false
	}
}
