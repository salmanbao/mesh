package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M79-monitoring-service/internal/ports"
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

type CreateAlertRuleInput struct {
	Name            string
	Query           string
	Threshold       float64
	DurationSeconds int
	Severity        string
	Enabled         bool
	Service         string
	Regex           string
}

type ListIncidentsInput struct {
	Status string
	Limit  int
}

type CreateSilenceInput struct {
	RuleID  string
	Reason  string
	StartAt time.Time
	EndAt   time.Time
}

type AuditQueryInput struct {
	ActorID    string
	ActionType string
	Limit      int
}

type Service struct {
	cfg Config

	rules      ports.AlertRuleRepository
	alerts     ports.AlertRepository
	incidents  ports.IncidentRepository
	silences   ports.SilenceRepository
	dashboards ports.DashboardRepository
	audits     ports.AuditRepository
	metrics    ports.MetricsRepository

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

	Rules      ports.AlertRuleRepository
	Alerts     ports.AlertRepository
	Incidents  ports.IncidentRepository
	Silences   ports.SilenceRepository
	Dashboards ports.DashboardRepository
	Audits     ports.AuditRepository
	Metrics    ports.MetricsRepository

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
		cfg.ServiceName = "M79-Monitoring-Service"
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
		rules:        deps.Rules,
		alerts:       deps.Alerts,
		incidents:    deps.Incidents,
		silences:     deps.Silences,
		dashboards:   deps.Dashboards,
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
