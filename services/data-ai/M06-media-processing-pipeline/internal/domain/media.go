package domain

import (
	"fmt"
	"strings"
	"time"
)

type AssetStatus string

type ApprovalStatus string

type JobType string

type JobStatus string

type OutputProfile string

type AspectRatio string

const (
	AssetStatusUploaded    AssetStatus = "uploaded"
	AssetStatusPendingScan AssetStatus = "pending_scan"
	AssetStatusProcessing  AssetStatus = "processing"
	AssetStatusCompleted   AssetStatus = "completed"
	AssetStatusFailed      AssetStatus = "failed"
)

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

const (
	JobTypeTranscode1080 JobType = "transcode_1080p"
	JobTypeTranscode720  JobType = "transcode_720p"
	JobTypeAspect916     JobType = "aspect_9_16"
	JobTypeAspect11      JobType = "aspect_1_1"
	JobTypeThumbnails    JobType = "thumbnails"
	JobTypeWatermark     JobType = "watermark"
)

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

const MaxJobAttempts = 3

const (
	Profile1080 OutputProfile = "1080p"
	Profile720  OutputProfile = "720p"
)

const (
	Aspect169 AspectRatio = "16:9"
	Aspect916 AspectRatio = "9:16"
	Aspect11  AspectRatio = "1:1"
)

type MediaAsset struct {
	AssetID          string         `json:"asset_id"`
	SubmissionID     string         `json:"submission_id"`
	OriginalFilename string         `json:"original_filename"`
	MIMEType         string         `json:"mime_type"`
	FileSize         int64          `json:"file_size"`
	DurationSeconds  int            `json:"duration_seconds"`
	SourceS3URL      string         `json:"source_s3_url"`
	Status           AssetStatus    `json:"status"`
	ApprovalStatus   ApprovalStatus `json:"approval_status"`
	ChecksumSHA256   string         `json:"checksum_sha256"`
	LastErrorCode    string         `json:"last_error_code,omitempty"`
	LastErrorMessage string         `json:"last_error_message,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type MediaJob struct {
	JobID        string    `json:"job_id"`
	AssetID      string    `json:"asset_id"`
	JobType      JobType   `json:"job_type"`
	Status       JobStatus `json:"status"`
	Attempts     int       `json:"attempts"`
	ErrorMessage string    `json:"error_message,omitempty"`
	QueuedAt     time.Time `json:"queued_at"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
}

type MediaOutput struct {
	OutputID    string        `json:"output_id"`
	AssetID     string        `json:"asset_id"`
	Profile     OutputProfile `json:"profile"`
	AspectRatio AspectRatio   `json:"aspect_ratio"`
	S3URL       string        `json:"s3_url"`
	CreatedAt   time.Time     `json:"created_at"`
}

type MediaThumbnail struct {
	ThumbnailID string      `json:"thumbnail_id"`
	AssetID     string      `json:"asset_id"`
	Position    string      `json:"position"`
	AspectRatio AspectRatio `json:"aspect_ratio"`
	S3URL       string      `json:"s3_url"`
	CreatedAt   time.Time   `json:"created_at"`
}

type WatermarkRecord struct {
	WatermarkID   string    `json:"watermark_id"`
	AssetID       string    `json:"asset_id"`
	WatermarkText string    `json:"watermark_text"`
	Placement     string    `json:"placement"`
	Opacity       float64   `json:"opacity"`
	AppliedAt     time.Time `json:"applied_at"`
}

type UploadInput struct {
	SubmissionID   string `json:"submission_id"`
	FileName       string `json:"file_name"`
	MIMEType       string `json:"mime_type"`
	FileSize       int64  `json:"file_size"`
	ChecksumSHA256 string `json:"checksum_sha256"`
}

func ValidateUploadInput(input UploadInput) error {
	if strings.TrimSpace(input.SubmissionID) == "" || strings.TrimSpace(input.FileName) == "" {
		return ErrInvalidInput
	}
	mime := strings.ToLower(strings.TrimSpace(input.MIMEType))
	if mime != "video/mp4" && mime != "video/quicktime" {
		return ErrInvalidInput
	}
	if input.FileSize <= 0 {
		return ErrInvalidInput
	}
	if input.FileSize > 500*1024*1024 {
		return ErrPayloadTooLarge
	}
	return nil
}

func NormalizeApprovalStatus(raw string) ApprovalStatus {
	status := ApprovalStatus(strings.ToLower(strings.TrimSpace(raw)))
	switch status {
	case ApprovalStatusApproved, ApprovalStatusRejected:
		return status
	default:
		return ApprovalStatusPending
	}
}

func NewDefaultJobs(assetID string, queuedAt time.Time) []MediaJob {
	jobTypes := []JobType{JobTypeTranscode1080, JobTypeTranscode720, JobTypeAspect916, JobTypeAspect11, JobTypeThumbnails, JobTypeWatermark}
	jobs := make([]MediaJob, 0, len(jobTypes))
	for idx, jt := range jobTypes {
		jobs = append(jobs, MediaJob{
			JobID:    fmt.Sprintf("%s-job-%d", assetID, idx+1),
			AssetID:  assetID,
			JobType:  jt,
			Status:   JobStatusQueued,
			QueuedAt: queuedAt,
		})
	}
	return jobs
}

func IsTerminal(status JobStatus) bool {
	return status == JobStatusCompleted || status == JobStatusFailed
}

func IsRetryableJobFailure(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "unsupported_codec", "malware_detected":
		return false
	default:
		return true
	}
}
