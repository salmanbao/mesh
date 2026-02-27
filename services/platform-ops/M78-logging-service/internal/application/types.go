package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/ports"
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

type IngestLogRecordInput struct {
	Timestamp  time.Time
	Level      string
	Service    string
	InstanceID string
	TraceID    string
	Message    string
	UserID     string
	ErrorCode  string
	Tags       map[string]any
}

type IngestLogsInput struct {
	Logs []IngestLogRecordInput
}

type SearchLogsInput struct {
	Service string
	Level   string
	From    *time.Time
	To      *time.Time
	Q       string
	Limit   int
}

type CreateExportInput struct {
	Query  map[string]any
	Format string
}

type CreateAlertRuleInput struct {
	Service   string
	Condition map[string]any
	Severity  string
	Enabled   bool
}

type AuditQueryInput struct {
	ActorID    string
	ActionType string
	Limit      int
}

type IngestResult struct {
	Ingested int
}

type ExportCreateResult struct {
	ExportID string
	Status   string
}

type Service struct {
	cfg Config

	logs    ports.LogEventRepository
	alerts  ports.AlertRuleRepository
	exp     ports.ExportRepository
	audits  ports.AuditRepository
	metrics ports.MetricsRepository

	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	ops          ports.OpsPublisher
	dlq          ports.DLQPublisher

	startedAt time.Time
	nowFn     func() time.Time
}

type Dependencies struct {
	Config Config

	Logs    ports.LogEventRepository
	Alerts  ports.AlertRuleRepository
	Exports ports.ExportRepository
	Audits  ports.AuditRepository
	Metrics ports.MetricsRepository

	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	Ops          ports.OpsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M78-Logging-Service"
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
		logs:         deps.Logs,
		alerts:       deps.Alerts,
		exp:          deps.Exports,
		audits:       deps.Audits,
		metrics:      deps.Metrics,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		ops:          deps.Ops,
		dlq:          deps.DLQ,
		startedAt:    now,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}
