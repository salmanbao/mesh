package contracts

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status            string `json:"status,omitempty"`
	Error             string `json:"error,omitempty"`
	Code              string `json:"code,omitempty"`
	Message           string `json:"message"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
}

type ReferrerMetric struct {
	Domain         string  `json:"domain,omitempty"`
	ReferrerDomain string  `json:"referrer_domain,omitempty"`
	Impressions    int     `json:"impressions"`
	Interactions   int     `json:"interactions,omitempty"`
	CTR            float64 `json:"ctr,omitempty"`
}

type ActionMetric struct {
	Action string `json:"action"`
	Count  int    `json:"count"`
}

type TrendPoint struct {
	Date         string  `json:"date"`
	Impressions  int     `json:"impressions"`
	Interactions int     `json:"interactions"`
	CTR          float64 `json:"ctr"`
}

type SettingsMetrics struct {
	TotalImpressions  int              `json:"total_impressions"`
	TotalInteractions int              `json:"total_interactions"`
	ClickThroughRate  float64          `json:"click_through_rate"`
	TopReferrers      []ReferrerMetric `json:"top_referrers"`
}

type EmbedSettingsResponse struct {
	EntityType         string          `json:"entity_type"`
	EntityID           string          `json:"entity_id"`
	AllowEmbedding     bool            `json:"allow_embedding"`
	DefaultTheme       string          `json:"default_theme"`
	PrimaryColor       string          `json:"primary_color"`
	CustomButtonText   string          `json:"custom_button_text"`
	AutoPlayVideo      bool            `json:"auto_play_video"`
	ShowCreatorInfo    bool            `json:"show_creator_info"`
	WhitelistedDomains []string        `json:"whitelisted_domains"`
	EmbedCode          string          `json:"embed_code"`
	Metrics            SettingsMetrics `json:"metrics"`
	UpdatedAt          string          `json:"updated_at,omitempty"`
}

type UpdateEmbedSettingsRequest struct {
	AllowEmbedding     *bool    `json:"allow_embedding,omitempty"`
	DefaultTheme       string   `json:"default_theme,omitempty"`
	PrimaryColor       string   `json:"primary_color,omitempty"`
	CustomButtonText   string   `json:"custom_button_text,omitempty"`
	AutoPlayVideo      *bool    `json:"auto_play_video,omitempty"`
	ShowCreatorInfo    *bool    `json:"show_creator_info,omitempty"`
	WhitelistedDomains []string `json:"whitelisted_domains,omitempty"`
}

type AnalyticsSummary struct {
	TotalImpressions  int            `json:"total_impressions"`
	TotalInteractions int            `json:"total_interactions"`
	ClickThroughRate  float64        `json:"click_through_rate"`
	TopActions        []ActionMetric `json:"top_actions"`
}

type EmbedAnalyticsResponse struct {
	Summary    AnalyticsSummary `json:"summary"`
	ByReferrer []ReferrerMetric `json:"by_referrer"`
	Trend      []TrendPoint     `json:"trend"`
}
