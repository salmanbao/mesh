package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/ports"
)

type Config struct {
	ServiceName          string
	Version              string
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

type CreatePolicyInput struct {
	PolicyID        string
	Scope           string
	TierFrom        string
	TierTo          string
	AfterDays       int
	LegalHoldExempt bool
}

type MoveToGlacierInput struct {
	FileID            string
	SubmissionID      string
	CampaignID        string
	SourceBucket      string
	SourceKey         string
	DestinationBucket string
	DestinationKey    string
	ChecksumMD5       string
	FileSizeBytes     int64
}

type ScheduleDeletionInput struct {
	CampaignID       string
	DeletionType     string
	DaysAfterClosure int
	FileIDs          []string
}

type AuditQueryInput struct {
	FileID     string
	CampaignID string
	Action     string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
}

type MetricObservation struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
}

type Service struct {
	cfg Config

	policies  ports.PolicyRepository
	lifecycle ports.LifecycleRepository
	batches   ports.DeletionBatchRepository
	audits    ports.AuditRepository
	metrics   ports.MetricsRepository

	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher

	startedAt time.Time
	nowFn     func() time.Time
}

type Dependencies struct {
	Config Config

	Policies  ports.PolicyRepository
	Lifecycle ports.LifecycleRepository
	Batches   ports.DeletionBatchRepository
	Audits    ports.AuditRepository
	Metrics   ports.MetricsRepository

	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M19-Storage-Lifecycle-Management"
	}
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
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
	now := time.Now().UTC()
	return &Service{
		cfg:          cfg,
		policies:     deps.Policies,
		lifecycle:    deps.Lifecycle,
		batches:      deps.Batches,
		audits:       deps.Audits,
		metrics:      deps.Metrics,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		startedAt:    now,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}
