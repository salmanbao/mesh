package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/domain"
)

type WarehouseRepository interface {
	UpsertUser(ctx context.Context, row domain.DimUser) error
	UpsertCampaign(ctx context.Context, row domain.DimCampaign) error
	AddSubmission(ctx context.Context, row domain.FactSubmission) error
	AddPayout(ctx context.Context, row domain.FactPayout) error
	AddTransaction(ctx context.Context, row domain.FactTransaction) error
	AddClick(ctx context.Context, row domain.FactClick) error
	UpsertDailyEarnings(ctx context.Context, row domain.DailyEarnings) error

	GetCreatorDashboard(ctx context.Context, userID string, from, to time.Time) (domain.CreatorDashboard, error)
	GetFinancialReport(ctx context.Context, from, to time.Time) (domain.FinancialReport, error)
}

type ExportRepository interface {
	Create(ctx context.Context, row domain.ExportJob) error
	Update(ctx context.Context, row domain.ExportJob) error
	GetByID(ctx context.Context, exportID string) (domain.ExportJob, error)
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}
