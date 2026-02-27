package application

import (
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/ports"
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

type PublishInput struct {
	EventID          string
	EventType        string
	CanonicalEvent   string
	OccurredAt       time.Time
	SourceService    string
	TraceID          string
	SchemaVersion    string
	PartitionKeyPath string
	PartitionKey     string
	Format           string
	Data             map[string]any
}

type CreateTopicInput struct {
	TopicName         string
	Partitions        int
	ReplicationFactor int
	RetentionDays     int
	CleanupPolicy     string
	CompressionType   string
}

type CreateACLInput struct {
	Principal    string
	ResourceType string
	ResourceName string
	PatternType  string
	Operations   []string
}

type RegisterSchemaInput struct {
	Subject       string
	SchemaType    string
	Compatibility string
	Schema        string
}

type ResetOffsetInput struct {
	GroupID   string
	Topic     string
	Partition int
	Offset    int64
	Reason    string
}

type DLQReplayInput struct {
	SourceTopic   string
	ConsumerGroup string
	ErrorType     string
	Limit         int
}

type DLQListInput struct {
	SourceTopic     string
	ConsumerGroup   string
	ErrorType       string
	Limit           int
	IncludeReplayed bool
}

type MetricObservation struct {
	Method     string
	Path       string
	StatusCode int
	Duration   time.Duration
}

type Service struct {
	cfg Config

	topics  ports.TopicRepository
	acls    ports.ACLRepository
	offsets ports.OffsetRepository
	schemas ports.SchemaRepository
	dlq     ports.DLQRepository
	metrics ports.MetricsRepository

	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlqPublisher ports.DLQPublisher

	startedAt time.Time
	nowFn     func() time.Time
}

type Dependencies struct {
	Config Config

	Topics  ports.TopicRepository
	ACLs    ports.ACLRepository
	Offsets ports.OffsetRepository
	Schemas ports.SchemaRepository
	DLQ     ports.DLQRepository
	Metrics ports.MetricsRepository

	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQPublisher ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M67-Event-Bus"
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
		topics:       deps.Topics,
		acls:         deps.ACLs,
		offsets:      deps.Offsets,
		schemas:      deps.Schemas,
		dlq:          deps.DLQ,
		metrics:      deps.Metrics,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlqPublisher: deps.DLQPublisher,
		startedAt:    now,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}
