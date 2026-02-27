package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/ports"
)

type Config struct {
	ServiceName          string
	EmbedBaseURL         string
	CacheTTL             time.Duration
	PerIPLimitPerHour    int
	PerEmbedLimitPerHour int
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type RenderEmbedInput struct {
	EntityType     string
	EntityID       string
	Theme          string
	Color          string
	ButtonText     string
	AutoPlay       bool
	Language       string
	Ref            string
	Referrer       string
	AcceptLanguage string
	UserAgent      string
	DNT            bool
	ClientIP       string
}

type UpdateEmbedSettingsInput struct {
	EntityType         string
	EntityID           string
	AllowEmbedding     *bool
	DefaultTheme       string
	PrimaryColor       string
	CustomButtonText   string
	AutoPlayVideo      *bool
	ShowCreatorInfo    *bool
	WhitelistedDomains []string
}

type AnalyticsQuery struct {
	EntityType  string
	EntityID    string
	StartDate   *time.Time
	EndDate     *time.Time
	Granularity string
	GroupBy     string
}

type ReferrerMetric struct {
	Domain       string
	Impressions  int
	Interactions int
	CTR          float64
}

type ActionMetric struct {
	Action string
	Count  int
}

type TrendPoint struct {
	Date         string
	Impressions  int
	Interactions int
	CTR          float64
}

type AnalyticsResult struct {
	TotalImpressions  int
	TotalInteractions int
	ClickThroughRate  float64
	TopActions        []ActionMetric
	ByReferrer        []ReferrerMetric
	Trend             []TrendPoint
}

type RenderedEmbed struct {
	HTML string
}

type Service struct {
	cfg          Config
	settings     ports.EmbedSettingsRepository
	cache        ports.EmbedCacheRepository
	impressions  ports.ImpressionRepository
	interactions ports.InteractionRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Settings     ports.EmbedSettingsRepository
	Cache        ports.EmbedCacheRepository
	Impressions  ports.ImpressionRepository
	Interactions ports.InteractionRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M66-Embed-Service"
	}
	if cfg.EmbedBaseURL == "" {
		cfg.EmbedBaseURL = "https://embed.platform.com"
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	if cfg.PerIPLimitPerHour <= 0 {
		cfg.PerIPLimitPerHour = 1000
	}
	if cfg.PerEmbedLimitPerHour <= 0 {
		cfg.PerEmbedLimitPerHour = 100
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.ConsumerPollInterval <= 0 {
		cfg.ConsumerPollInterval = 2 * time.Second
	}
	return &Service{cfg: cfg, settings: deps.Settings, cache: deps.Cache, impressions: deps.Impressions, interactions: deps.Interactions, idempotency: deps.Idempotency, eventDedup: deps.EventDedup, nowFn: func() time.Time { return time.Now().UTC() }}
}
