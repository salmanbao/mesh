package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/domain"
)

type WarehouseRepository interface {
	CreateReferralEvent(ctx context.Context, row domain.ReferralEvent) error
	UpsertDailyAggregate(ctx context.Context, row domain.ReferralAggregateDaily) error
	UpsertFunnelAggregate(ctx context.Context, row domain.ReferralFunnelAggregate) error
	UpsertCohortRetention(ctx context.Context, row domain.ReferralCohortRetention) error
	UpsertGeoAggregate(ctx context.Context, row domain.ReferralGeoAggregate) error
	GetFunnel(ctx context.Context, from, to time.Time) (domain.FunnelReport, error)
	GetLeaderboard(ctx context.Context, period string, now time.Time) (domain.LeaderboardReport, error)
	GetCohortRetention(ctx context.Context, from, to time.Time) (domain.CohortRetentionReport, error)
	GetGeo(ctx context.Context, from, to time.Time) (domain.GeoPerformanceReport, error)
	GetPayoutForecast(ctx context.Context, period string, now time.Time) (domain.PayoutForecast, error)
}

type ExportJobRepository interface {
	Create(ctx context.Context, row domain.ReferralExportJob) error
	Update(ctx context.Context, row domain.ReferralExportJob) error
	GetByID(ctx context.Context, id string) (domain.ReferralExportJob, error)
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

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
