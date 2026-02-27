package domain

import (
	"strings"
	"time"
)

const (
	PolicyStatusActive = "active"

	TierStandard           = "STANDARD"
	TierGlacier            = "GLACIER"
	TierGlacierDeepArchive = "GLACIER_DEEP_ARCHIVE"

	FileStatusUploaded     = "uploaded"
	FileStatusApproved     = "approved"
	FileStatusArchivedCold = "archived_cold"
	FileStatusSoftDeleted  = "soft_deleted"
	FileStatusHardDeleted  = "hard_deleted"
	FileStatusMoveFailed   = "move_failed"

	DeletionTypeRawFiles = "raw_files"
)

type StoragePolicy struct {
	PolicyID        string    `json:"policy_id"`
	Scope           string    `json:"scope"`
	TierFrom        string    `json:"tier_from"`
	TierTo          string    `json:"tier_to"`
	AfterDays       int       `json:"after_days"`
	LegalHoldExempt bool      `json:"legal_hold_exempt"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type LifecycleFile struct {
	FileID            string    `json:"file_id"`
	CampaignID        string    `json:"campaign_id,omitempty"`
	SubmissionID      string    `json:"submission_id,omitempty"`
	FileSizeBytes     int64     `json:"file_size_bytes,omitempty"`
	SourceBucket      string    `json:"source_bucket,omitempty"`
	SourceKey         string    `json:"source_key,omitempty"`
	DestinationBucket string    `json:"destination_bucket,omitempty"`
	DestinationKey    string    `json:"destination_key,omitempty"`
	ChecksumMD5       string    `json:"checksum_md5,omitempty"`
	StorageTier       string    `json:"storage_tier"`
	Status            string    `json:"status"`
	LegalHold         bool      `json:"legal_hold"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DeletionBatch struct {
	BatchID          string    `json:"batch_id"`
	CampaignID       string    `json:"campaign_id"`
	DeletionType     string    `json:"deletion_type"`
	FileIDs          []string  `json:"file_ids"`
	DaysAfterClosure int       `json:"days_after_closure"`
	FileCount        int       `json:"file_count"`
	ScheduledFor     time.Time `json:"scheduled_for"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type AuditRecord struct {
	AuditID       string    `json:"audit_id"`
	FileID        string    `json:"file_id,omitempty"`
	CampaignID    string    `json:"campaign_id,omitempty"`
	Action        string    `json:"action"`
	TriggeredBy   string    `json:"triggered_by,omitempty"`
	FileSizeBytes int64     `json:"file_size_bytes,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	InitiatedAt   time.Time `json:"initiated_at"`
	CompletedAt   time.Time `json:"completed_at"`
}

type AuditQuery struct {
	FileID     string
	CampaignID string
	Action     string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
}

type AuditQueryResult struct {
	Records           []AuditRecord `json:"records"`
	TotalFilesDeleted int64         `json:"total_files_deleted"`
	TotalSizeFreed    int64         `json:"total_size_freed"`
}

type LifecycleJob struct {
	JobID     string    `json:"job_id"`
	FileID    string    `json:"file_id,omitempty"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AnalyticsSummary struct {
	TotalObjects int64            `json:"total_objects"`
	ByTier       map[string]int64 `json:"by_tier"`
	MonthlyCost  float64          `json:"monthly_cost"`
	LastRunAt    time.Time        `json:"last_run_at"`
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

func IsValidTier(v string) bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case TierStandard, TierGlacier, TierGlacierDeepArchive:
		return true
	default:
		return false
	}
}

func IsValidPolicyStatus(v string) bool {
	return strings.TrimSpace(v) == PolicyStatusActive
}

func IsValidFileStatus(v string) bool {
	switch strings.TrimSpace(v) {
	case FileStatusUploaded, FileStatusApproved, FileStatusArchivedCold, FileStatusSoftDeleted, FileStatusHardDeleted, FileStatusMoveFailed:
		return true
	default:
		return false
	}
}
