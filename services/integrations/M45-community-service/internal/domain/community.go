package domain

import (
	"strings"
	"time"
)

type Platform string

const (
	PlatformDiscord  Platform = "discord"
	PlatformSlack    Platform = "slack"
	PlatformTelegram Platform = "telegram"
	PlatformInternal Platform = "internal"
)

func IsValidPlatform(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(PlatformDiscord), string(PlatformSlack), string(PlatformTelegram), string(PlatformInternal):
		return true
	default:
		return false
	}
}

type IntegrationStatus string

const (
	IntegrationStatusActive       IntegrationStatus = "active"
	IntegrationStatusError        IntegrationStatus = "error"
	IntegrationStatusDisconnected IntegrationStatus = "disconnected"
)

type GrantStatus string

const (
	GrantStatusActive  GrantStatus = "active"
	GrantStatusPending GrantStatus = "pending"
	GrantStatusRevoked GrantStatus = "revoked"
	GrantStatusFailed  GrantStatus = "failed"
)

type HealthStatus string

const (
	HealthStatusHealthy      HealthStatus = "healthy"
	HealthStatusError        HealthStatus = "error"
	HealthStatusTimeout      HealthStatus = "timeout"
	HealthStatusRateLimited  HealthStatus = "rate_limited"
	HealthStatusTokenExpired HealthStatus = "token_expired"
)

type CommunityIntegration struct {
	IntegrationID string            `json:"integration_id"`
	CreatorID     string            `json:"creator_id"`
	Platform      string            `json:"platform"`
	CommunityName string            `json:"community_name"`
	Config        map[string]string `json:"config"`
	Status        IntegrationStatus `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	LastSyncAt    *time.Time        `json:"last_sync_at,omitempty"`
}

type ProductCommunityMapping struct {
	MappingID     string            `json:"mapping_id"`
	ProductID     string            `json:"product_id"`
	IntegrationID string            `json:"integration_id"`
	Tier          string            `json:"tier"`
	RoleConfig    map[string]string `json:"role_config,omitempty"`
	Enabled       bool              `json:"enabled"`
	CreatedAt     time.Time         `json:"created_at"`
}

type CommunityGrant struct {
	GrantID          string      `json:"grant_id"`
	UserID           string      `json:"user_id"`
	ProductID        string      `json:"product_id"`
	IntegrationID    string      `json:"integration_id"`
	OrderID          string      `json:"order_id"`
	Tier             string      `json:"tier"`
	Status           GrantStatus `json:"status"`
	GrantedAt        time.Time   `json:"granted_at"`
	RevokedAt        *time.Time  `json:"revoked_at,omitempty"`
	RevocationReason string      `json:"revocation_reason,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type CommunityAuditLog struct {
	AuditLogID    string            `json:"audit_log_id"`
	Timestamp     time.Time         `json:"timestamp"`
	ActionType    string            `json:"action_type"`
	UserID        string            `json:"user_id,omitempty"`
	PerformedBy   string            `json:"performed_by,omitempty"`
	PerformerRole string            `json:"performer_role,omitempty"`
	IntegrationID string            `json:"integration_id,omitempty"`
	ProductID     string            `json:"product_id,omitempty"`
	GrantID       string            `json:"grant_id,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	Outcome       string            `json:"outcome"`
	ErrorMessage  string            `json:"error_message,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type CommunityHealthCheck struct {
	HealthCheckID  string       `json:"health_check_id"`
	IntegrationID  string       `json:"integration_id"`
	CheckedAt      time.Time    `json:"checked_at"`
	Status         HealthStatus `json:"status"`
	LatencyMS      int          `json:"latency_ms"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	HTTPStatusCode int          `json:"http_status_code,omitempty"`
}
