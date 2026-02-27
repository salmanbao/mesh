package domain

import "time"

const (
	EntityTypeWhop     = "whop"
	EntityTypeCampaign = "campaign"
	EntityTypeApp      = "app"
	EntityTypeClip     = "clip"
)

func IsValidEntityType(v string) bool {
	switch v {
	case EntityTypeWhop, EntityTypeCampaign, EntityTypeApp, EntityTypeClip:
		return true
	default:
		return false
	}
}

type EmbedSettings struct {
	ID                 string    `json:"id"`
	EntityType         string    `json:"entity_type"`
	EntityID           string    `json:"entity_id"`
	AllowEmbedding     bool      `json:"allow_embedding"`
	DefaultTheme       string    `json:"default_theme"`
	PrimaryColor       string    `json:"primary_color"`
	CustomButtonText   string    `json:"custom_button_text"`
	AutoPlayVideo      bool      `json:"auto_play_video"`
	ShowCreatorInfo    bool      `json:"show_creator_info"`
	WhitelistedDomains []string  `json:"whitelisted_domains"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          string    `json:"updated_by"`
}

type EmbedCache struct {
	CacheKey   string    `json:"cache_key"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	HTML       string    `json:"html"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type Impression struct {
	ID               string    `json:"id"`
	EntityType       string    `json:"entity_type"`
	EntityID         string    `json:"entity_id"`
	ReferrerDomain   string    `json:"referrer_domain"`
	UserAgentBrowser string    `json:"user_agent_browser"`
	IPAnonymized     string    `json:"ip_anonymized"`
	DNTEnabled       bool      `json:"dnt_enabled"`
	ThemeUsed        string    `json:"theme_used"`
	CustomColor      string    `json:"custom_color"`
	OccurredAt       time.Time `json:"occurred_at"`
}

type Interaction struct {
	ID             string            `json:"id"`
	EntityType     string            `json:"entity_type"`
	EntityID       string            `json:"entity_id"`
	Action         string            `json:"action"`
	ReferrerDomain string            `json:"referrer_domain"`
	ImpressionID   string            `json:"impression_id"`
	Metadata       map[string]string `json:"metadata"`
	OccurredAt     time.Time         `json:"occurred_at"`
}
