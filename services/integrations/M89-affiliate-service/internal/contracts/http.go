package contracts

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message"`
}

type CreateReferralLinkRequest struct {
	Channel     string `json:"channel"`
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
}

type CreateReferralLinkResponse struct {
	LinkID string `json:"link_id"`
	URL    string `json:"url"`
}

type DashboardResponse struct {
	AffiliateID       string    `json:"affiliate_id"`
	TotalReferrals    int       `json:"total_referrals"`
	TotalClicks       int       `json:"total_clicks"`
	TotalAttributions int       `json:"total_attributions"`
	ConversionRate    float64   `json:"conversion_rate"`
	PendingEarnings   float64   `json:"pending_earnings"`
	PaidEarnings      float64   `json:"paid_earnings"`
	TopLinks          []TopLink `json:"top_links"`
}

type TopLink struct {
	LinkID  string `json:"link_id"`
	Clicks  int    `json:"clicks"`
	Channel string `json:"channel"`
}

type EarningResponse struct {
	EarningID string  `json:"earning_id"`
	OrderID   string  `json:"order_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
}

type EarningsListResponse struct {
	Items []EarningResponse `json:"items"`
}

type ExportRequest struct {
	Format string `json:"format,omitempty"`
}

type ExportResponse struct {
	ExportID string `json:"export_id"`
	Status   string `json:"status"`
}

type SuspendAffiliateRequest struct {
	Reason string `json:"reason"`
}

type SuspendAffiliateResponse struct {
	AffiliateID string `json:"affiliate_id"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at"`
}

type ManualAttributionRequest struct {
	OrderID      string  `json:"order_id"`
	ConversionID string  `json:"conversion_id"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	ClickID      string  `json:"click_id,omitempty"`
}

type ManualAttributionResponse struct {
	AttributionID string  `json:"attribution_id"`
	AffiliateID   string  `json:"affiliate_id"`
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	AttributedAt  string  `json:"attributed_at"`
}
