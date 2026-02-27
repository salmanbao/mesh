package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/domain"
)

type PolicyRepository interface {
	Create(ctx context.Context, row domain.StoragePolicy) error
	List(ctx context.Context) ([]domain.StoragePolicy, error)
}

type LifecycleRepository interface {
	Upsert(ctx context.Context, row domain.LifecycleFile) error
	GetByID(ctx context.Context, fileID string) (domain.LifecycleFile, error)
	ListByCampaign(ctx context.Context, campaignID string) ([]domain.LifecycleFile, error)
	AnalyticsSummary(ctx context.Context) (domain.AnalyticsSummary, error)
}

type DeletionBatchRepository interface {
	Create(ctx context.Context, row domain.DeletionBatch) error
	GetByID(ctx context.Context, batchID string) (domain.DeletionBatch, error)
	List(ctx context.Context, limit int) ([]domain.DeletionBatch, error)
}

type AuditRepository interface {
	Create(ctx context.Context, row domain.AuditRecord) error
	Query(ctx context.Context, q domain.AuditQuery) (domain.AuditQueryResult, error)
}

type MetricsRepository interface {
	IncCounter(ctx context.Context, name string, labels map[string]string, delta float64) error
	ObserveHistogram(ctx context.Context, name string, labels map[string]string, value float64, buckets []float64) error
	Snapshot(ctx context.Context) (MetricsSnapshot, error)
}

type MetricCounterPoint struct {
	Name   string
	Labels map[string]string
	Value  float64
}

type MetricHistogramPoint struct {
	Name    string
	Labels  map[string]string
	Buckets map[string]float64
	Sum     float64
	Count   float64
}

type MetricsSnapshot struct {
	Counters   []MetricCounterPoint
	Histograms []MetricHistogramPoint
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
