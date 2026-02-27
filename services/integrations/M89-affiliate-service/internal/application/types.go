package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/ports"
)

type Config struct {
	ServiceName          string
	PublicBaseURL        string
	ReferralCookieTTL    time.Duration
	CommissionRate       float64
	PayoutThreshold      float64
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateReferralLinkInput struct {
	Channel     string
	UTMSource   string
	UTMMedium   string
	UTMCampaign string
}

type TrackClickInput struct {
	Token       string
	ReferrerURL string
	ClientIP    string
	UserAgent   string
	CookieID    string
}

type TrackClickResult struct {
	RedirectURL string
	CookieID    string
	AffiliateID string
	LinkID      string
}

type Dashboard struct {
	AffiliateID       string
	TotalReferrals    int
	TotalClicks       int
	TotalAttributions int
	ConversionRate    float64
	PendingEarnings   float64
	PaidEarnings      float64
	TopLinks          []TopLinkMetric
}

type TopLinkMetric struct {
	LinkID  string
	Clicks  int
	Channel string
}

type ExportInput struct {
	Format string
}

type ExportResult struct {
	ExportID string
	Status   string
}

type SuspendAffiliateInput struct {
	AffiliateID string
	Reason      string
}

type RecordAttributionInput struct {
	AffiliateID  string
	ClickID      string
	OrderID      string
	ConversionID string
	Amount       float64
	Currency     string
}

type Service struct {
	cfg Config

	affiliates   ports.AffiliateRepository
	links        ports.ReferralLinkRepository
	clicks       ports.ReferralClickRepository
	attributions ports.ReferralAttributionRepository
	earnings     ports.AffiliateEarningRepository
	payouts      ports.AffiliatePayoutRepository
	auditLogs    ports.AffiliateAuditLogRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	outbox       ports.OutboxRepository

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher

	nowFn func() time.Time
}

type Dependencies struct {
	Config Config

	Affiliates   ports.AffiliateRepository
	Links        ports.ReferralLinkRepository
	Clicks       ports.ReferralClickRepository
	Attributions ports.ReferralAttributionRepository
	Earnings     ports.AffiliateEarningRepository
	Payouts      ports.AffiliatePayoutRepository
	AuditLogs    ports.AffiliateAuditLogRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M89-Affiliate-Service"
	}
	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = "https://platform.com"
	}
	if cfg.ReferralCookieTTL <= 0 {
		cfg.ReferralCookieTTL = 30 * 24 * time.Hour
	}
	if cfg.CommissionRate <= 0 {
		cfg.CommissionRate = 0.10
	}
	if cfg.PayoutThreshold <= 0 {
		cfg.PayoutThreshold = 0.50
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
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	return &Service{cfg: cfg, affiliates: deps.Affiliates, links: deps.Links, clicks: deps.Clicks, attributions: deps.Attributions, earnings: deps.Earnings, payouts: deps.Payouts, auditLogs: deps.AuditLogs, idempotency: deps.Idempotency, eventDedup: deps.EventDedup, outbox: deps.Outbox, domainEvents: deps.DomainEvents, analytics: deps.Analytics, dlq: deps.DLQ, nowFn: func() time.Time { return time.Now().UTC() }}
}
