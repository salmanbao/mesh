package application

import (
	"time"

	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	OutboxFlushBatchSize int
	RecommendationTTL    time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type GetRecommendationsInput struct {
	Role    string
	Limit   int
	Segment string
}

type FeedbackInput struct {
	EventID          string
	EventType        string
	OccurredAt       string
	SourceService    string
	TraceID          string
	SchemaVersion    string
	PartitionKeyPath string
	PartitionKey     string
	EntityID         string
}

type OverrideInput struct {
	OverrideType string
	EntityID     string
	Scope        string
	ScopeValue   string
	Multiplier   float64
	Reason       string
	EndDate      string
}

type Service struct {
	cfg Config

	recommendations ports.RecommendationsRepository
	feedback        ports.FeedbackRepository
	overrides       ports.OverridesRepository
	models          ports.ModelsRepository
	abTests         ports.ABTestRepository
	idempotency     ports.IdempotencyRepository
	eventDedup      ports.EventDedupRepository
	outbox          ports.OutboxRepository

	campaignDiscovery ports.CampaignDiscoveryReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config Config

	Recommendations ports.RecommendationsRepository
	Feedback        ports.FeedbackRepository
	Overrides       ports.OverridesRepository
	Models          ports.ModelsRepository
	ABTests         ports.ABTestRepository
	Idempotency     ports.IdempotencyRepository
	EventDedup      ports.EventDedupRepository
	Outbox          ports.OutboxRepository

	CampaignDiscovery ports.CampaignDiscoveryReader

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M58-Content-Recommendation-Engine"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	if cfg.RecommendationTTL <= 0 {
		cfg.RecommendationTTL = time.Hour
	}
	return &Service{
		cfg:               cfg,
		recommendations:   deps.Recommendations,
		feedback:          deps.Feedback,
		overrides:         deps.Overrides,
		models:            deps.Models,
		abTests:           deps.ABTests,
		idempotency:       deps.Idempotency,
		eventDedup:        deps.EventDedup,
		outbox:            deps.Outbox,
		campaignDiscovery: deps.CampaignDiscovery,
		domainEvents:      deps.DomainEvents,
		analytics:         deps.Analytics,
		dlq:               deps.DLQ,
		nowFn:             time.Now().UTC,
	}
}

func normalizeRole(raw string) string {
	if role := domain.NormalizeRole(raw); role != "" {
		return role
	}
	return domain.RoleClipper
}
