package application

import (
	"time"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	DashboardCacheTTL    time.Duration
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type DashboardQueryInput struct {
	ViewID     string
	DateRange  string
	FromDate   string
	ToDate     string
	DeviceType string
	Timezone   string
}

type SaveLayoutInput struct {
	DeviceType string
	Items      []LayoutItemInput
}

type LayoutItemInput struct {
	WidgetID string
	Position int
	Visible  bool
	Size     string
}

type CreateCustomViewInput struct {
	ViewName         string
	WidgetIDs        []string
	DateRangeDefault string
	SetAsDefault     bool
}

type Service struct {
	cfg Config

	layouts       ports.LayoutRepository
	views         ports.CustomViewRepository
	preferences   ports.UserPreferenceRepository
	invalidations ports.CacheInvalidationRepository
	cache         ports.DashboardCacheRepository
	idempotency   ports.IdempotencyRepository
	eventDedup    ports.EventDedupRepository
	outbox        ports.OutboxRepository

	profile      ports.ProfileReader
	billing      ports.BillingReader
	content      ports.ContentReader
	escrow       ports.EscrowReader
	onboarding   ports.OnboardingReader
	finance      ports.FinanceReader
	rewards      ports.RewardReader
	gamification ports.GamificationReader
	analytics    ports.AnalyticsReader
	products     ports.ProductReader

	dlq   ports.DLQPublisher
	nowFn func() time.Time
}

type Dependencies struct {
	Config Config

	Layouts       ports.LayoutRepository
	Views         ports.CustomViewRepository
	Preferences   ports.UserPreferenceRepository
	Invalidations ports.CacheInvalidationRepository
	Cache         ports.DashboardCacheRepository
	Idempotency   ports.IdempotencyRepository
	EventDedup    ports.EventDedupRepository
	Outbox        ports.OutboxRepository

	Profile      ports.ProfileReader
	Billing      ports.BillingReader
	Content      ports.ContentReader
	Escrow       ports.EscrowReader
	Onboarding   ports.OnboardingReader
	Finance      ports.FinanceReader
	Rewards      ports.RewardReader
	Gamification ports.GamificationReader
	Analytics    ports.AnalyticsReader
	Products     ports.ProductReader

	DLQ ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M55-Dashboard-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.DashboardCacheTTL <= 0 {
		cfg.DashboardCacheTTL = 5 * time.Minute
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	return &Service{
		cfg:           cfg,
		layouts:       deps.Layouts,
		views:         deps.Views,
		preferences:   deps.Preferences,
		invalidations: deps.Invalidations,
		cache:         deps.Cache,
		idempotency:   deps.Idempotency,
		eventDedup:    deps.EventDedup,
		outbox:        deps.Outbox,
		profile:       deps.Profile,
		billing:       deps.Billing,
		content:       deps.Content,
		escrow:        deps.Escrow,
		onboarding:    deps.Onboarding,
		finance:       deps.Finance,
		rewards:       deps.Rewards,
		gamification:  deps.Gamification,
		analytics:     deps.Analytics,
		products:      deps.Products,
		dlq:           deps.DLQ,
		nowFn:         time.Now().UTC,
	}
}
