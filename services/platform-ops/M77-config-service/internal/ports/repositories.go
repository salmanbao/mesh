package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/domain"
)

type ConfigKeyRepository interface {
	GetByName(ctx context.Context, keyName string) (domain.ConfigKey, error)
	Upsert(ctx context.Context, row domain.ConfigKey) (domain.ConfigKey, error)
	Update(ctx context.Context, row domain.ConfigKey) error
	List(ctx context.Context) ([]domain.ConfigKey, error)
}

type ConfigValueRepository interface {
	Upsert(ctx context.Context, row domain.ConfigValue) (domain.ConfigValue, error)
	Get(ctx context.Context, keyID, environment, serviceScope string) (domain.ConfigValue, error)
	ListByEnvironment(ctx context.Context, environment string) ([]domain.ConfigValue, error)
}

type ConfigVersionRepository interface {
	Create(ctx context.Context, row domain.ConfigVersion) error
	ListByScope(ctx context.Context, keyID, environment, serviceScope string, limit int) ([]domain.ConfigVersion, error)
	GetByVersionNumber(ctx context.Context, keyID, environment, serviceScope string, versionNumber int) (domain.ConfigVersion, error)
	NextVersionNumber(ctx context.Context, keyID string) (int, error)
}

type RolloutRuleRepository interface {
	UpsertForKey(ctx context.Context, row domain.RolloutRule) (domain.RolloutRule, error)
	GetByKeyID(ctx context.Context, keyID string) (domain.RolloutRule, error)
	List(ctx context.Context) ([]domain.RolloutRule, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, row domain.AuditLog) error
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
