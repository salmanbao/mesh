package domain

import "time"

type Affiliate struct {
	AffiliateID    string    `json:"affiliate_id"`
	UserID         string    `json:"user_id"`
	Status         string    `json:"status"`
	DefaultRate    float64   `json:"default_rate"`
	BalancePending float64   `json:"balance_pending"`
	BalancePaid    float64   `json:"balance_paid"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ReferralLink struct {
	LinkID         string    `json:"link_id"`
	AffiliateID    string    `json:"affiliate_id"`
	Token          string    `json:"token"`
	Channel        string    `json:"channel"`
	UTMSource      string    `json:"utm_source"`
	UTMMedium      string    `json:"utm_medium"`
	UTMCampaign    string    `json:"utm_campaign"`
	DestinationURL string    `json:"destination_url"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReferralClick struct {
	ClickID       string    `json:"click_id"`
	LinkID        string    `json:"link_id"`
	AffiliateID   string    `json:"affiliate_id"`
	ReferrerURL   string    `json:"referrer_url"`
	IPHash        string    `json:"ip_hash"`
	UserAgentHash string    `json:"user_agent_hash"`
	CookieID      string    `json:"cookie_id"`
	ClickedAt     time.Time `json:"clicked_at"`
}

type ReferralAttribution struct {
	AttributionID string    `json:"attribution_id"`
	AffiliateID   string    `json:"affiliate_id"`
	ClickID       string    `json:"click_id"`
	ConversionID  string    `json:"conversion_id"`
	OrderID       string    `json:"order_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	AttributedAt  time.Time `json:"attributed_at"`
}

type AffiliateEarning struct {
	EarningID     string    `json:"earning_id"`
	AffiliateID   string    `json:"affiliate_id"`
	AttributionID string    `json:"attribution_id"`
	OrderID       string    `json:"order_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AffiliatePayout struct {
	PayoutID    string    `json:"payout_id"`
	AffiliateID string    `json:"affiliate_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	QueuedAt    time.Time `json:"queued_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AffiliateAuditLog struct {
	AuditLogID  string            `json:"audit_log_id"`
	AffiliateID string            `json:"affiliate_id"`
	Action      string            `json:"action"`
	ActorID     string            `json:"actor_id"`
	Reason      string            `json:"reason"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
}
