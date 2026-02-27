package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/ports"
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
	IPAddress      string
	UserAgent      string
}

type MetricObservation struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
}

type GetConfigInput struct {
	Environment  string
	ServiceScope string
	UserID       string
	Role         string
	Tier         string
}

type PatchConfigInput struct {
	Key          string
	Environment  string
	ServiceScope string
	ValueType    string
	Value        any
}

type ImportConfigEntry struct {
	Key       string
	ValueType string
	Value     any
}

type ImportConfigInput struct {
	Environment  string
	ServiceScope string
	Entries      []ImportConfigEntry
}

type ExportConfigInput struct {
	Environment  string
	ServiceScope string
}

type RollbackConfigInput struct {
	Key          string
	Environment  string
	ServiceScope string
	Version      int
}

type CreateRolloutRuleInput struct {
	Key        string
	RuleType   string
	Percentage int
	Role       string
	Tier       string
}

type AuditQueryInput struct {
	KeyName      string
	Environment  string
	ServiceScope string
	ActorID      string
	Limit        int
}

type PatchResult struct {
	Key          string
	Environment  string
	ServiceScope string
	Version      int
}

type RollbackResult struct {
	PatchResult
	RolledBackTo int
}

type Service struct {
	cfg Config

	keys    ports.ConfigKeyRepository
	values  ports.ConfigValueRepository
	vers    ports.ConfigVersionRepository
	rules   ports.RolloutRuleRepository
	audits  ports.AuditLogRepository
	metrics ports.MetricsRepository

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

	Keys     ports.ConfigKeyRepository
	Values   ports.ConfigValueRepository
	Versions ports.ConfigVersionRepository
	Rules    ports.RolloutRuleRepository
	Audits   ports.AuditLogRepository
	Metrics  ports.MetricsRepository

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
		cfg.ServiceName = "M77-Config-Service"
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
		keys:         deps.Keys,
		values:       deps.Values,
		vers:         deps.Versions,
		rules:        deps.Rules,
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
