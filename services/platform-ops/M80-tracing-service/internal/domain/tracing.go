package domain

import (
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

const (
	ExportStatusQueued    = "queued"
	ExportStatusRunning   = "running"
	ExportStatusCompleted = "completed"
	ExportStatusFailed    = "failed"
)

type TraceRecord struct {
	TraceID     string    `json:"trace_id"`
	RootService string    `json:"root_service"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	DurationMS  int64     `json:"duration_ms"`
	Error       bool      `json:"error"`
	Environment string    `json:"environment,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SpanRecord struct {
	SpanID         string    `json:"span_id"`
	TraceID        string    `json:"trace_id"`
	ParentSpanID   string    `json:"parent_span_id,omitempty"`
	ServiceName    string    `json:"service_name"`
	OperationName  string    `json:"operation_name"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	DurationMS     int64     `json:"duration_ms"`
	Error          bool      `json:"error"`
	HTTPStatusCode int       `json:"http_status_code,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type SpanTag struct {
	TagID     string    `json:"tag_id"`
	SpanID    string    `json:"span_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type SamplingPolicy struct {
	PolicyID        string    `json:"policy_id"`
	ServiceName     string    `json:"service_name"`
	RuleType        string    `json:"rule_type"`
	Probability     *float64  `json:"probability,omitempty"`
	MaxTracesPerMin *int      `json:"max_traces_per_min,omitempty"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ExportJob struct {
	ExportID     string            `json:"export_id"`
	RequestedBy  string            `json:"requested_by"`
	Status       string            `json:"status"`
	TraceID      string            `json:"trace_id,omitempty"`
	Filters      map[string]string `json:"filters,omitempty"`
	Format       string            `json:"format"`
	OutputURI    string            `json:"output_uri,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type AuditLog struct {
	AuditID     string            `json:"audit_id"`
	ActorUserID string            `json:"actor_user_id,omitempty"`
	Action      string            `json:"action"`
	TargetType  string            `json:"target_type,omitempty"`
	TargetID    string            `json:"target_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	OccurredAt  time.Time         `json:"occurred_at"`
}

type IngestedSpan struct {
	TraceID        string            `json:"trace_id"`
	SpanID         string            `json:"span_id"`
	ParentSpanID   string            `json:"parent_span_id,omitempty"`
	ServiceName    string            `json:"service_name"`
	OperationName  string            `json:"operation_name"`
	StartTime      time.Time         `json:"start_time"`
	EndTime        time.Time         `json:"end_time"`
	Error          bool              `json:"error"`
	HTTPStatusCode int               `json:"http_status_code,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	Environment    string            `json:"environment,omitempty"`
}

type TraceSearchQuery struct {
	ServiceName  string
	ErrorOnly    *bool
	DurationGTMS *int64
	TraceID      string
	Limit        int
}

type TraceSearchHit struct {
	TraceID    string `json:"trace_id"`
	DurationMS int64  `json:"duration_ms"`
	Error      bool   `json:"error"`
}

type TraceDetail struct {
	Trace TraceRecord  `json:"trace"`
	Spans []SpanRecord `json:"spans"`
	Tags  []SpanTag    `json:"tags,omitempty"`
}

type MetricsSnapshot struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
	IngestedSpans   int64 `json:"ingested_spans_total"`
	StoredTraces    int64 `json:"stored_traces"`
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

func IsHexTraceID(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) != 32 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}

func IsHexSpanID(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) != 16 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}

func IsValidSamplingRuleType(v string) bool {
	switch strings.TrimSpace(v) {
	case "always_sample_errors", "slow_traces", "probabilistic", "rate_limited":
		return true
	default:
		return false
	}
}

func IsValidExportStatus(v string) bool {
	switch strings.TrimSpace(v) {
	case ExportStatusQueued, ExportStatusRunning, ExportStatusCompleted, ExportStatusFailed:
		return true
	default:
		return false
	}
}

func NormalizeSpansForTimeline(spans []SpanRecord) []SpanRecord {
	out := append([]SpanRecord(nil), spans...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartTime.Equal(out[j].StartTime) {
			return out[i].SpanID < out[j].SpanID
		}
		return out[i].StartTime.Before(out[j].StartTime)
	})
	return out
}
