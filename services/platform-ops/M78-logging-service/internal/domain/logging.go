package domain

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"

	ExportFormatCSV  = "csv"
	ExportFormatJSON = "json"

	ExportStatusPending = "pending"
	ExportStatusReady   = "ready"
	ExportStatusFailed  = "failed"

	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

type LogEvent struct {
	EventID    string          `json:"event_id"`
	Timestamp  time.Time       `json:"timestamp"`
	Level      string          `json:"level"`
	Service    string          `json:"service"`
	InstanceID string          `json:"instance_id,omitempty"`
	TraceID    string          `json:"trace_id,omitempty"`
	Message    string          `json:"message"`
	UserID     string          `json:"user_id,omitempty"`
	ErrorCode  string          `json:"error_code,omitempty"`
	Tags       json.RawMessage `json:"tags,omitempty"`
	Redacted   bool            `json:"redacted"`
	IngestedAt time.Time       `json:"ingested_at"`
}

type LogSearchQuery struct {
	Service string
	Level   string
	From    *time.Time
	To      *time.Time
	Q       string
	Limit   int
}

type LogExport struct {
	ExportID    string          `json:"export_id"`
	RequestedBy string          `json:"requested_by"`
	Query       json.RawMessage `json:"query"`
	Format      string          `json:"format"`
	Status      string          `json:"status"`
	FileURL     string          `json:"file_url,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

type AlertRule struct {
	RuleID    string          `json:"rule_id"`
	Service   string          `json:"service"`
	Condition json.RawMessage `json:"condition"`
	Severity  string          `json:"severity"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type AuditLog struct {
	AuditID    string          `json:"audit_id"`
	ActorID    string          `json:"actor_id"`
	ActionType string          `json:"action_type"`
	ActionAt   time.Time       `json:"action_at"`
	IPAddress  string          `json:"ip_address,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

type AuditQuery struct {
	ActorID    string
	ActionType string
	Limit      int
}

type AuditQueryResult struct {
	Logs []AuditLog `json:"logs"`
}

type HealthReport struct {
	Status        string                    `json:"status"`
	Timestamp     time.Time                 `json:"timestamp"`
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Version       string                    `json:"version"`
	Checks        map[string]ComponentCheck `json:"checks"`
}

type ComponentCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LatencyMS   int       `json:"latency_ms"`
	LastChecked time.Time `json:"last_checked"`
}

type MetricsSnapshot struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}

func IsValidLogLevel(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal:
		return true
	default:
		return false
	}
}

func NormalizeLogLevel(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func IsValidExportFormat(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ExportFormatCSV, ExportFormatJSON:
		return true
	default:
		return false
	}
}

func NormalizeExportFormat(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func IsValidSeverity(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func NormalizeSeverity(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
