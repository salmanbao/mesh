package application

import (
	"time"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/ports"
)

type Config struct {
	ServiceName    string
	IdempotencyTTL time.Duration
	EventDedupTTL  time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type DashboardInput struct {
	UserID   string
	DateFrom string
	DateTo   string
}

type FinancialReportInput struct {
	DateFrom string
	DateTo   string
}

type ExportInput struct {
	ReportType string
	Format     string
	DateFrom   string
	DateTo     string
	Filters    map[string]string
}

type Service struct {
	cfg Config

	warehouse   ports.WarehouseRepository
	exports     ports.ExportRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository

	voting     ports.VotingReader
	social     ports.SocialReader
	tracking   ports.TrackingReader
	submission ports.SubmissionReader
	finance    ports.FinanceReader

	dlq   ports.DLQPublisher
	nowFn func() time.Time
}

type Dependencies struct {
	Config Config

	Warehouse   ports.WarehouseRepository
	Exports     ports.ExportRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository

	Voting     ports.VotingReader
	Social     ports.SocialReader
	Tracking   ports.TrackingReader
	Submission ports.SubmissionReader
	Finance    ports.FinanceReader

	DLQ ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M54-Analytics-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:         cfg,
		warehouse:   deps.Warehouse,
		exports:     deps.Exports,
		idempotency: deps.Idempotency,
		eventDedup:  deps.EventDedup,
		voting:      deps.Voting,
		social:      deps.Social,
		tracking:    deps.Tracking,
		submission:  deps.Submission,
		finance:     deps.Finance,
		dlq:         deps.DLQ,
		nowFn:       time.Now().UTC,
	}
}

func normalizeRole(raw string) string {
	switch raw {
	case "admin":
		return "admin"
	default:
		return "creator"
	}
}

func coalesceUserID(actor Actor, requested string) string {
	if requested != "" {
		return requested
	}
	return actor.SubjectID
}

func makeExportResponse(job domain.ExportJob) map[string]interface{} {
	return map[string]interface{}{
		"export_id":    job.ExportID,
		"status":       job.Status,
		"download_url": job.DownloadURL,
		"created_at":   job.CreatedAt,
		"ready_at":     job.ReadyAt,
	}
}
