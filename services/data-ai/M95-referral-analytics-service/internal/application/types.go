package application

import (
	"strings"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	OutboxFlushBatchSize int
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type DateRangeInput struct {
	StartDate string
	EndDate   string
}

type LeaderboardInput struct {
	Period string
}

type CohortInput struct {
	CohortStart string
	CohortEnd   string
}

type ExportInput struct {
	ExportType string
	Period     string
	Format     string
	Filters    map[string]string
}

type Service struct {
	cfg Config

	warehouse   ports.WarehouseRepository
	exports     ports.ExportJobRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	affiliate ports.AffiliateReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config Config

	Warehouse   ports.WarehouseRepository
	Exports     ports.ExportJobRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository

	Affiliate ports.AffiliateReader

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M95-Referral-Analytics-Service"
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
	return &Service{
		cfg:          cfg,
		warehouse:    deps.Warehouse,
		exports:      deps.Exports,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		affiliate:    deps.Affiliate,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}

func normalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin":
		return "admin"
	case "analyst":
		return "analyst"
	default:
		return "user"
	}
}

func MakeExportResponse(job domain.ReferralExportJob) map[string]interface{} {
	return map[string]interface{}{
		"id":           job.ID,
		"status":       job.Status,
		"export_type":  job.ExportType,
		"format":       job.Format,
		"period":       job.Period,
		"output_uri":   job.OutputURI,
		"created_at":   job.CreatedAt,
		"completed_at": job.CompletedAt,
	}
}
