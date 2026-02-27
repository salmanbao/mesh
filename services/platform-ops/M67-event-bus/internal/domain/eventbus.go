package domain

import (
	"regexp"
	"strings"
	"time"
)

const (
	TopicStatusActive    = "active"
	TopicStatusPending   = "pending"
	ACLStatusActive      = "active"
	ACLStatusPending     = "pending"
	SchemaTypeJSON       = "json"
	SchemaTypeAvro       = "avro"
	CleanupDelete        = "delete"
	CleanupCompact       = "compact"
	CleanupCompactDelete = "compact,delete"
	CompressionSnappy    = "snappy"
	CompressionZstd      = "zstd"
)

var topicNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z0-9_]+(\.[a-z0-9_]+)?$`)

var canonicalDomains = map[string]struct{}{
	"submission":   {},
	"payout":       {},
	"campaign":     {},
	"user":         {},
	"transaction":  {},
	"reward":       {},
	"dispute":      {},
	"moderation":   {},
	"analytics":    {},
	"embed":        {},
	"notification": {},
	"audit":        {},
	"compliance":   {},
	"legal":        {},
}

var deprecatedPluralDomains = map[string]struct{}{
	"submissions": {},
	"payouts":     {},
	"campaigns":   {},
	"users":       {},
}

type PublishedEvent struct {
	EventID          string         `json:"event_id"`
	EventType        string         `json:"event_type"`
	CanonicalEvent   string         `json:"canonical_event,omitempty"`
	OccurredAt       time.Time      `json:"occurred_at"`
	SourceService    string         `json:"source_service"`
	TraceID          string         `json:"trace_id"`
	SchemaVersion    string         `json:"schema_version"`
	PartitionKeyPath string         `json:"partition_key_path"`
	PartitionKey     string         `json:"partition_key"`
	Format           string         `json:"format"`
	Data             map[string]any `json:"data"`
}

type PublishResult struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	Status     string    `json:"status"`
	Format     string    `json:"format"`
	AcceptedAt time.Time `json:"accepted_at"`
	Topic      string    `json:"topic"`
}

type Topic struct {
	ID                string    `json:"id"`
	TopicName         string    `json:"topic_name"`
	Partitions        int       `json:"partitions"`
	ReplicationFactor int       `json:"replication_factor"`
	RetentionDays     int       `json:"retention_days"`
	CleanupPolicy     string    `json:"cleanup_policy"`
	CompressionType   string    `json:"compression_type"`
	Status            string    `json:"status"`
	CreatedBy         string    `json:"created_by,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type ACLRecord struct {
	ID           string    `json:"id"`
	Principal    string    `json:"principal"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	PatternType  string    `json:"pattern_type"`
	Operations   []string  `json:"operations"`
	Status       string    `json:"status"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ConsumerOffsetAudit struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	Topic     string    `json:"topic"`
	Partition int       `json:"partition"`
	Offset    int64     `json:"offset"`
	Reason    string    `json:"reason,omitempty"`
	ChangedBy string    `json:"changed_by,omitempty"`
	ChangedAt time.Time `json:"changed_at"`
}

type SchemaRecord struct {
	ID            string    `json:"id"`
	Subject       string    `json:"subject"`
	SchemaType    string    `json:"schema_type"`
	Compatibility string    `json:"compatibility"`
	Schema        string    `json:"schema"`
	Version       int       `json:"version"`
	CreatedBy     string    `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type DLQMessage struct {
	ID            string         `json:"id"`
	SourceTopic   string         `json:"source_topic"`
	ConsumerGroup string         `json:"consumer_group,omitempty"`
	ErrorType     string         `json:"error_type,omitempty"`
	ErrorSummary  string         `json:"error_summary"`
	RetryCount    int            `json:"retry_count"`
	EventID       string         `json:"event_id,omitempty"`
	OriginalEvent map[string]any `json:"original_event"`
	CreatedAt     time.Time      `json:"created_at"`
	ReplayedAt    *time.Time     `json:"replayed_at,omitempty"`
}

type DLQReplayResult struct {
	Requested int       `json:"requested"`
	Replayed  int       `json:"replayed"`
	Failed    int       `json:"failed"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

type DLQQuery struct {
	SourceTopic     string
	ConsumerGroup   string
	ErrorType       string
	Limit           int
	IncludeReplayed bool
}

type MetricsSnapshot struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}

type ComponentCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LatencyMS   int       `json:"latency_ms,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

type HealthReport struct {
	Status        string                    `json:"status"`
	Timestamp     time.Time                 `json:"timestamp"`
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Version       string                    `json:"version,omitempty"`
	Checks        map[string]ComponentCheck `json:"checks"`
}

func IsValidTopicName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 || !topicNameRE.MatchString(name) {
		return false
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}
	if _, ok := canonicalDomains[parts[0]]; ok {
		return true
	}
	_, deprecated := deprecatedPluralDomains[parts[0]]
	return deprecated
}

func IsDeprecatedPluralTopic(name string) bool {
	parts := strings.Split(strings.TrimSpace(name), ".")
	if len(parts) < 2 {
		return false
	}
	_, ok := deprecatedPluralDomains[parts[0]]
	return ok
}

func IsValidFormat(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case SchemaTypeJSON, SchemaTypeAvro:
		return true
	default:
		return false
	}
}

func IsValidSchemaType(v string) bool { return IsValidFormat(v) }

func IsValidCompatibility(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "BACKWARD", "BACKWARD_TRANSITIVE", "FULL", "NONE":
		return true
	default:
		return false
	}
}

func IsValidCleanupPolicy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case CleanupDelete, CleanupCompact, CleanupCompactDelete:
		return true
	default:
		return false
	}
}

func IsValidCompression(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case CompressionSnappy, CompressionZstd:
		return true
	default:
		return false
	}
}

func TimestampWithinSkew(ts, now time.Time, skew time.Duration) bool {
	if ts.IsZero() {
		return false
	}
	diff := now.Sub(ts)
	if diff < 0 {
		diff = -diff
	}
	return diff <= skew
}
