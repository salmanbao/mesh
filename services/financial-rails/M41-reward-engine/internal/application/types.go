package application

import (
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/ports"
)

type Config struct {
	ServiceName                  string
	IdempotencyTTL               time.Duration
	EventDedupTTL                time.Duration
	OutboxFlushBatchSize         int
	MinimumPayoutThreshold       float64
	MaxRolloverBalance           float64
	FraudRejectThreshold         float64
	DefaultRatePer1K             float64
	EnableDomainEventConsumption bool
	EnablePayoutEligibleEmission bool
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CalculateRewardInput struct {
	UserID                  string
	SubmissionID            string
	CampaignID              string
	LockedViews             int64
	RatePer1K               float64
	FraudScore              float64
	VerificationCompletedAt time.Time
	EventID                 string
}

type Service struct {
	cfg         Config
	rewards     ports.RewardRepository
	rollovers   ports.RolloverRepository
	snapshots   ports.SnapshotRepository
	audit       ports.AuditLogRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	auth       ports.AuthReader
	campaign   ports.CampaignRateReader
	voting     ports.VotingReader
	tracking   ports.TrackingReader
	submission ports.SubmissionReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Rewards      ports.RewardRepository
	Rollovers    ports.RolloverRepository
	Snapshots    ports.SnapshotRepository
	Audit        ports.AuditLogRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository
	Auth         ports.AuthReader
	Campaign     ports.CampaignRateReader
	Voting       ports.VotingReader
	Tracking     ports.TrackingReader
	Submission   ports.SubmissionReader
	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M41-Reward-Engine"
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
	if cfg.MinimumPayoutThreshold <= 0 {
		cfg.MinimumPayoutThreshold = 0.5
	}
	if cfg.MaxRolloverBalance <= 0 {
		cfg.MaxRolloverBalance = 50
	}
	if cfg.FraudRejectThreshold <= 0 {
		cfg.FraudRejectThreshold = 0.70
	}
	if cfg.DefaultRatePer1K <= 0 {
		cfg.DefaultRatePer1K = 2.5
	}
	return &Service{
		cfg:          cfg,
		rewards:      deps.Rewards,
		rollovers:    deps.Rollovers,
		snapshots:    deps.Snapshots,
		audit:        deps.Audit,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		auth:         deps.Auth,
		campaign:     deps.Campaign,
		voting:       deps.Voting,
		tracking:     deps.Tracking,
		submission:   deps.Submission,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		nowFn:        time.Now().UTC,
	}
}

func (s *Service) EnsureRewardEligible(_ string, _ string, _ float64) error {
	return nil
}

type RewardHistoryOutput struct {
	Items []domain.Reward
	Total int
}
