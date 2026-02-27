package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type PublishEventRequest struct {
	EventID          string         `json:"event_id"`
	EventType        string         `json:"event_type"`
	CanonicalEvent   string         `json:"canonical_event,omitempty"`
	OccurredAt       string         `json:"occurred_at,omitempty"`
	Timestamp        string         `json:"timestamp,omitempty"`
	SourceService    string         `json:"source_service"`
	TraceID          string         `json:"trace_id"`
	SchemaVersion    string         `json:"schema_version"`
	PartitionKeyPath string         `json:"partition_key_path"`
	PartitionKey     string         `json:"partition_key"`
	Format           string         `json:"format,omitempty"`
	Data             map[string]any `json:"data"`
}

type PublishEventResponse struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	Topic      string `json:"topic"`
	Status     string `json:"status"`
	Format     string `json:"format"`
	AcceptedAt string `json:"accepted_at"`
}

type CreateTopicRequest struct {
	TopicName         string `json:"topic_name"`
	Partitions        int    `json:"partitions"`
	ReplicationFactor int    `json:"replication_factor"`
	RetentionDays     int    `json:"retention_days,omitempty"`
	CleanupPolicy     string `json:"cleanup_policy,omitempty"`
	CompressionType   string `json:"compression_type,omitempty"`
}

type TopicResponse struct {
	ID                string `json:"id"`
	TopicName         string `json:"topic_name"`
	Partitions        int    `json:"partitions"`
	ReplicationFactor int    `json:"replication_factor"`
	RetentionDays     int    `json:"retention_days"`
	CleanupPolicy     string `json:"cleanup_policy"`
	CompressionType   string `json:"compression_type"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
}

type CreateACLRequest struct {
	Principal    string   `json:"principal"`
	ResourceType string   `json:"resource_type"`
	ResourceName string   `json:"resource_name"`
	PatternType  string   `json:"pattern_type,omitempty"`
	Operations   []string `json:"operations"`
}

type ACLResponse struct {
	ID           string   `json:"id"`
	Principal    string   `json:"principal"`
	ResourceType string   `json:"resource_type"`
	ResourceName string   `json:"resource_name"`
	PatternType  string   `json:"pattern_type"`
	Operations   []string `json:"operations"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
}

type RegisterSchemaRequest struct {
	Subject       string `json:"subject"`
	SchemaType    string `json:"schema_type,omitempty"`
	Compatibility string `json:"compatibility,omitempty"`
	Schema        string `json:"schema"`
}

type SchemaResponse struct {
	ID            string `json:"id"`
	Subject       string `json:"subject"`
	SchemaType    string `json:"schema_type"`
	Compatibility string `json:"compatibility"`
	Version       int    `json:"version"`
	CreatedAt     string `json:"created_at"`
}

type ResetOffsetRequest struct {
	Topic     string `json:"topic"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Reason    string `json:"reason,omitempty"`
}

type OffsetAuditResponse struct {
	ID        string `json:"id"`
	GroupID   string `json:"group_id"`
	Topic     string `json:"topic"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Reason    string `json:"reason,omitempty"`
	ChangedAt string `json:"changed_at"`
}

type ReplayDLQRequest struct {
	SourceTopic   string `json:"source_topic,omitempty"`
	ConsumerGroup string `json:"consumer_group,omitempty"`
	ErrorType     string `json:"error_type,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type ReplayDLQResponse struct {
	Requested int    `json:"requested"`
	Replayed  int    `json:"replayed"`
	Failed    int    `json:"failed"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}

type DLQMessageResponse struct {
	ID            string `json:"id"`
	SourceTopic   string `json:"source_topic"`
	ConsumerGroup string `json:"consumer_group,omitempty"`
	ErrorType     string `json:"error_type,omitempty"`
	ErrorSummary  string `json:"error_summary"`
	RetryCount    int    `json:"retry_count"`
	EventID       string `json:"event_id,omitempty"`
	CreatedAt     string `json:"created_at"`
	ReplayedAt    string `json:"replayed_at,omitempty"`
}

type CacheMetricsResponse struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}
