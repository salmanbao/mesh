package contracts

import "time"

type DashboardQuery struct {
	ViewID     string `json:"view_id,omitempty"`
	DateRange  string `json:"date_range,omitempty"`
	FromDate   string `json:"from_date,omitempty"`
	ToDate     string `json:"to_date,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	Timezone   string `json:"timezone,omitempty"`
}

type LayoutItemRequest struct {
	WidgetID string `json:"widget_id"`
	Position int    `json:"position"`
	Visible  bool   `json:"visible"`
	Size     string `json:"size"`
}

type SaveLayoutRequest struct {
	DeviceType string              `json:"device_type"`
	Widgets    []LayoutItemRequest `json:"widgets"`
}

type CreateCustomViewRequest struct {
	ViewName         string   `json:"view_name"`
	WidgetIDs        []string `json:"widget_ids"`
	DateRangeDefault string   `json:"date_range_default"`
	SetAsDefault     bool     `json:"set_as_default"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

type DashboardResponse struct {
	Dashboard interface{} `json:"dashboard"`
}

type CreateCustomViewResponse struct {
	ViewID   string `json:"view_id"`
	ViewName string `json:"view_name"`
}

type EnvelopeMetadata struct {
	TraceID string `json:"trace_id,omitempty"`
}

type EventEnvelope struct {
	EventID          string      `json:"event_id"`
	EventType        string      `json:"event_type"`
	EventClass       string      `json:"event_class,omitempty"`
	OccurredAt       time.Time   `json:"occurred_at"`
	PartitionKeyPath string      `json:"partition_key_path"`
	PartitionKey     string      `json:"partition_key"`
	SourceService    string      `json:"source_service"`
	SchemaVersion    string      `json:"schema_version"`
	Metadata         interface{} `json:"metadata,omitempty"`
	Data             interface{} `json:"data"`
}

type DLQRecord struct {
	OriginalEvent EventEnvelope `json:"original_event"`
	ErrorSummary  string        `json:"error_summary"`
	RetryCount    int           `json:"retry_count"`
	FirstSeenAt   time.Time     `json:"first_seen_at"`
	LastErrorAt   time.Time     `json:"last_error_at"`
	SourceTopic   string        `json:"source_topic,omitempty"`
	TraceID       string        `json:"trace_id,omitempty"`
}
