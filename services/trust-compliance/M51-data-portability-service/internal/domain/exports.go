package domain

import "time"

const (
	ExportRequestTypeExport = "export"
	ExportRequestTypeErase  = "erase"
)

const (
	ExportStatusPending    = "pending"
	ExportStatusProcessing = "processing"
	ExportStatusCompleted  = "completed"
	ExportStatusFailed     = "failed"
)

const (
	EventExportCompleted = "export.completed"
	EventExportFailed    = "export.failed"
)

type ExportRequest struct {
	RequestID    string     `json:"request_id"`
	UserID       string     `json:"user_id"`
	RequestType  string     `json:"request_type"`
	Format       string     `json:"format,omitempty"`
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	RequestedAt  time.Time  `json:"requested_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DownloadURL  string     `json:"download_url,omitempty"`
	FailureCause string     `json:"failure_cause,omitempty"`
}

type AuditLog struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type"`
	RequestID  string            `json:"request_id"`
	UserID     string            `json:"user_id"`
	ActorID    string            `json:"actor_id"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
