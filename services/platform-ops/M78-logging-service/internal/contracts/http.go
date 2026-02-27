package contracts

import "encoding/json"

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

type IngestLogRecord struct {
	Timestamp  string         `json:"timestamp"`
	Level      string         `json:"level"`
	Service    string         `json:"service"`
	InstanceID string         `json:"instance_id,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	Message    string         `json:"message"`
	UserID     string         `json:"user_id,omitempty"`
	ErrorCode  string         `json:"error_code,omitempty"`
	Tags       map[string]any `json:"tags,omitempty"`
}

type IngestLogsRequest struct {
	Logs []IngestLogRecord `json:"logs"`
}

type IngestLogsResponse struct {
	Ingested int `json:"ingested"`
}

type SearchLogItem struct {
	EventID    string          `json:"event_id"`
	Timestamp  string          `json:"timestamp"`
	Level      string          `json:"level"`
	Service    string          `json:"service"`
	InstanceID string          `json:"instance_id,omitempty"`
	TraceID    string          `json:"trace_id,omitempty"`
	Message    string          `json:"message"`
	UserID     string          `json:"user_id,omitempty"`
	ErrorCode  string          `json:"error_code,omitempty"`
	Tags       json.RawMessage `json:"tags,omitempty"`
	Redacted   bool            `json:"redacted"`
	IngestedAt string          `json:"ingested_at"`
}

type SearchLogsResponse struct {
	Items []SearchLogItem `json:"items"`
}

type CreateExportRequest struct {
	Query  map[string]any `json:"query"`
	Format string         `json:"format"`
}

type CreateExportResponse struct {
	ExportID string `json:"export_id"`
	Status   string `json:"status"`
}

type ExportDetailResponse struct {
	ExportID    string         `json:"export_id"`
	RequestedBy string         `json:"requested_by"`
	Query       map[string]any `json:"query,omitempty"`
	Format      string         `json:"format"`
	Status      string         `json:"status"`
	FileURL     string         `json:"file_url,omitempty"`
	CreatedAt   string         `json:"created_at"`
	CompletedAt string         `json:"completed_at,omitempty"`
}

type CreateAlertRuleRequest struct {
	Service   string         `json:"service"`
	Condition map[string]any `json:"condition"`
	Severity  string         `json:"severity"`
	Enabled   bool           `json:"enabled"`
}

type AlertRuleResponse struct {
	RuleID    string         `json:"rule_id"`
	Service   string         `json:"service"`
	Condition map[string]any `json:"condition,omitempty"`
	Severity  string         `json:"severity"`
	Enabled   bool           `json:"enabled"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type ListAlertRulesResponse struct {
	Items []AlertRuleResponse `json:"items"`
}

type AuditLogItem struct {
	AuditID    string          `json:"audit_id"`
	ActorID    string          `json:"actor_id"`
	ActionType string          `json:"action_type"`
	ActionAt   string          `json:"action_at"`
	IPAddress  string          `json:"ip_address,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

type AuditQueryResponse struct {
	Logs []AuditLogItem `json:"logs"`
}

type ServiceMetricsResponse struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}
