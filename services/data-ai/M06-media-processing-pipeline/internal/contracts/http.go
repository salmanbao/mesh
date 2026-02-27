package contracts

import "time"

type UploadRequest struct {
	SubmissionID   string `json:"submission_id"`
	FileName       string `json:"file_name"`
	MIMEType       string `json:"mime_type"`
	FileSize       int64  `json:"file_size"`
	ChecksumSHA256 string `json:"checksum_sha256,omitempty"`
}

type UploadResponse struct {
	AssetID   string `json:"asset_id"`
	UploadURL string `json:"upload_url"`
	ExpiresIn int    `json:"expires_in"`
}

type OutputDTO struct {
	Profile     string `json:"profile"`
	AspectRatio string `json:"aspect_ratio"`
	URL         string `json:"url"`
}

type ThumbnailDTO struct {
	Position    string `json:"position"`
	AspectRatio string `json:"aspect_ratio"`
	URL         string `json:"url"`
}

type AssetStatusResponse struct {
	AssetID    string         `json:"asset_id"`
	Status     string         `json:"status"`
	Outputs    []OutputDTO    `json:"outputs"`
	Thumbnails []ThumbnailDTO `json:"thumbnails"`
	ErrorCode  string         `json:"error_code,omitempty"`
	Error      string         `json:"error,omitempty"`
}

type RetryResponse struct {
	AssetID       string `json:"asset_id"`
	JobsRestarted int    `json:"jobs_restarted"`
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

type EventEnvelope struct {
	EventID          string      `json:"event_id"`
	EventType        string      `json:"event_type"`
	EventClass       string      `json:"event_class,omitempty"`
	OccurredAt       time.Time   `json:"occurred_at"`
	PartitionKeyPath string      `json:"partition_key_path"`
	PartitionKey     string      `json:"partition_key"`
	SourceService    string      `json:"source_service"`
	TraceID          string      `json:"trace_id"`
	SchemaVersion    string      `json:"schema_version"`
	Metadata         interface{} `json:"metadata,omitempty"`
	Data             interface{} `json:"data"`
}
