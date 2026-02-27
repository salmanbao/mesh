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

type IngestSpan struct {
	TraceID        string            `json:"trace_id"`
	SpanID         string            `json:"span_id"`
	ParentSpanID   string            `json:"parent_span_id,omitempty"`
	ServiceName    string            `json:"service_name"`
	OperationName  string            `json:"operation_name"`
	StartTime      string            `json:"start_time"`
	EndTime        string            `json:"end_time"`
	Error          bool              `json:"error"`
	HTTPStatusCode int               `json:"http_status_code,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	Environment    string            `json:"environment,omitempty"`
}

type IngestRequest struct {
	Spans []IngestSpan `json:"spans"`
}

type TraceSearchItem struct {
	TraceID    string `json:"trace_id"`
	DurationMS int64  `json:"duration_ms"`
	Error      bool   `json:"error"`
}

type SpanDetailResponse struct {
	SpanID       string            `json:"span_id"`
	ServiceName  string            `json:"service_name"`
	Operation    string            `json:"operation_name"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	DurationMS   int64             `json:"duration_ms"`
	Error        bool              `json:"error"`
	Tags         map[string]string `json:"tags,omitempty"`
}

type TraceDetailResponse struct {
	TraceID string               `json:"trace_id"`
	Spans   []SpanDetailResponse `json:"spans"`
}

type CreateSamplingPolicyRequest struct {
	ServiceName     string   `json:"service_name"`
	RuleType        string   `json:"rule_type"`
	Probability     *float64 `json:"probability,omitempty"`
	MaxTracesPerMin *int     `json:"max_traces_per_min,omitempty"`
}

type CreateSamplingPolicyResponse struct {
	PolicyID string `json:"policy_id"`
}

type SamplingPolicyResponse struct {
	PolicyID        string   `json:"policy_id"`
	ServiceName     string   `json:"service_name"`
	RuleType        string   `json:"rule_type"`
	Probability     *float64 `json:"probability,omitempty"`
	MaxTracesPerMin *int     `json:"max_traces_per_min,omitempty"`
	Enabled         bool     `json:"enabled"`
}

type CreateExportRequest struct {
	TraceID string            `json:"trace_id,omitempty"`
	Format  string            `json:"format,omitempty"`
	Filters map[string]string `json:"filters,omitempty"`
}

type ExportResponse struct {
	ExportID     string `json:"export_id"`
	Status       string `json:"status"`
	OutputURI    string `json:"output_uri,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type CacheMetricsResponse struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}
