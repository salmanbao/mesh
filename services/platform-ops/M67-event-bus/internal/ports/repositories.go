package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/domain"
)

type TopicRepository interface {
	Create(ctx context.Context, row domain.Topic) error
	GetByName(ctx context.Context, topicName string) (domain.Topic, error)
	List(ctx context.Context, limit int) ([]domain.Topic, error)
}

type ACLRepository interface {
	Create(ctx context.Context, row domain.ACLRecord) error
	List(ctx context.Context, limit int) ([]domain.ACLRecord, error)
}

type OffsetRepository interface {
	Create(ctx context.Context, row domain.ConsumerOffsetAudit) error
	ListByGroup(ctx context.Context, groupID string, limit int) ([]domain.ConsumerOffsetAudit, error)
}

type SchemaRepository interface {
	Register(ctx context.Context, row domain.SchemaRecord) (domain.SchemaRecord, error)
	GetLatestBySubject(ctx context.Context, subject string) (domain.SchemaRecord, error)
	List(ctx context.Context, limit int) ([]domain.SchemaRecord, error)
}

type DLQRepository interface {
	Create(ctx context.Context, row domain.DLQMessage) error
	Query(ctx context.Context, q domain.DLQQuery) ([]domain.DLQMessage, error)
	MarkReplayed(ctx context.Context, ids []string, at time.Time) error
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
