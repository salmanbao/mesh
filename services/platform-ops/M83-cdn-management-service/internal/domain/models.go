package domain

import "time"

type CDNConfig struct {
	ConfigID     string            `json:"config_id"`
	Provider     string            `json:"provider"`
	Version      int               `json:"version"`
	Status       string            `json:"status"`
	Config       map[string]any    `json:"config"`
	TLSVersion   string            `json:"tls_version"`
	HeaderRules  map[string]string `json:"header_rules,omitempty"`
	SignedURLTTL int               `json:"signed_url_ttl_seconds,omitempty"`
	CreatedBy    string            `json:"created_by"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type PurgeRequest struct {
	PurgeID     string    `json:"purge_id"`
	Scope       string    `json:"scope"`
	Target      string    `json:"target"`
	Status      string    `json:"status"`
	CompletedIn int       `json:"completed_in_seconds"`
	RequestedBy string    `json:"requested_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type Certificate struct {
	CertID        string    `json:"cert_id"`
	Provider      string    `json:"provider"`
	Domain        string    `json:"domain"`
	ExpiresAt     time.Time `json:"expires_at"`
	AutoRenew     bool      `json:"auto_renew"`
	TLSVersion    string    `json:"tls_version"`
	LastRenewedAt time.Time `json:"last_renewed_at,omitempty"`
}

type Metrics struct {
	HitRate         float64 `json:"hit_rate"`
	BandwidthGB     float64 `json:"bandwidth_gb"`
	EgressCostUSD   float64 `json:"egress_cost_usd"`
	P95LatencyMS    float64 `json:"p95_latency_ms"`
	ErrorRate       float64 `json:"error_rate"`
	OriginHealthy   bool    `json:"origin_healthy"`
	PendingPurges   int     `json:"pending_purges"`
	ActiveConfigVer int     `json:"active_config_version"`
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Response    []byte
	ExpiresAt   time.Time
}
