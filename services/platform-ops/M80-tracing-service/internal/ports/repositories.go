package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/domain"
)

type TraceRepository interface {
	UpsertFromSpans(ctx context.Context, spans []domain.SpanRecord, environment string) ([]domain.TraceRecord, error)
	GetByID(ctx context.Context, traceID string) (domain.TraceRecord, error)
	Search(ctx context.Context, q domain.TraceSearchQuery) ([]domain.TraceSearchHit, error)
	Count(ctx context.Context) (int, error)
}

type SpanRepository interface {
	UpsertBatch(ctx context.Context, spans []domain.SpanRecord) (inserted int, duplicates int, err error)
	ListByTraceID(ctx context.Context, traceID string) ([]domain.SpanRecord, error)
}

type SpanTagRepository interface {
	ReplaceForSpans(ctx context.Context, tags []domain.SpanTag) error
	ListByTraceID(ctx context.Context, traceID string) ([]domain.SpanTag, error)
}

type SamplingPolicyRepository interface {
	Create(ctx context.Context, row domain.SamplingPolicy) error
	List(ctx context.Context) ([]domain.SamplingPolicy, error)
	GetByID(ctx context.Context, policyID string) (domain.SamplingPolicy, error)
}

type ExportRepository interface {
	Create(ctx context.Context, row domain.ExportJob) error
	GetByID(ctx context.Context, exportID string) (domain.ExportJob, error)
	Update(ctx context.Context, row domain.ExportJob) error
	List(ctx context.Context, limit int) ([]domain.ExportJob, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, row domain.AuditLog) error
	List(ctx context.Context, limit int) ([]domain.AuditLog, error)
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
