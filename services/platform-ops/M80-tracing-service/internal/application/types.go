package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/ports"
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

type IngestInput struct {
	Format string
	Spans  []domain.IngestedSpan
}

type IngestResult struct {
	Accepted   int
	Duplicates int
	Rejected   int
}

type SearchInput struct {
	TraceID      string
	ServiceName  string
	ErrorOnly    *bool
	DurationGTMS *int64
	Limit        int
}

type CreateSamplingPolicyInput struct {
	ServiceName     string
	RuleType        string
	Probability     *float64
	MaxTracesPerMin *int
}

type CreateExportInput struct {
	TraceID string
	Format  string
	Filters map[string]string
}

type MetricObservation struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
}

type Service struct {
	cfg Config

	traces    ports.TraceRepository
	spans     ports.SpanRepository
	tags      ports.SpanTagRepository
	policies  ports.SamplingPolicyRepository
	exports   ports.ExportRepository
	auditLogs ports.AuditLogRepository
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

	Traces    ports.TraceRepository
	Spans     ports.SpanRepository
	Tags      ports.SpanTagRepository
	Policies  ports.SamplingPolicyRepository
	Exports   ports.ExportRepository
	AuditLogs ports.AuditLogRepository
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
		cfg.ServiceName = "M80-Tracing-Service"
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
		traces:       deps.Traces,
		spans:        deps.Spans,
		tags:         deps.Tags,
		policies:     deps.Policies,
		exports:      deps.Exports,
		auditLogs:    deps.AuditLogs,
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
