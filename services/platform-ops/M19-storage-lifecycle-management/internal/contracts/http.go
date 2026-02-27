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

type CreateStoragePolicyRequest struct {
	PolicyID        string `json:"policy_id,omitempty"`
	Scope           string `json:"scope"`
	TierFrom        string `json:"tier_from"`
	TierTo          string `json:"tier_to"`
	AfterDays       int    `json:"after_days"`
	LegalHoldExempt bool   `json:"legal_hold_exempt"`
}

type CreateStoragePolicyResponse struct {
	PolicyID  string `json:"policy_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type AnalyticsSummaryResponse struct {
	TotalObjects int64            `json:"total_objects"`
	ByTier       map[string]int64 `json:"by_tier"`
	MonthlyCost  float64          `json:"monthly_cost"`
	LastRunAt    string           `json:"last_run_at,omitempty"`
}

type MoveToGlacierRequest struct {
	FileID            string `json:"file_id"`
	SubmissionID      string `json:"submission_id,omitempty"`
	CampaignID        string `json:"campaign_id,omitempty"`
	SourceBucket      string `json:"source_bucket"`
	SourceKey         string `json:"source_key"`
	DestinationBucket string `json:"destination_bucket"`
	DestinationKey    string `json:"destination_key"`
	ChecksumMD5       string `json:"checksum_md5,omitempty"`
	FileSizeBytes     int64  `json:"file_size_bytes,omitempty"`
}

type MoveToGlacierResponse struct {
	JobID   string `json:"job_id"`
	FileID  string `json:"file_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ScheduleDeletionRequest struct {
	CampaignID       string   `json:"campaign_id"`
	DeletionType     string   `json:"deletion_type"`
	DaysAfterClosure int      `json:"days_after_closure"`
	FileIDs          []string `json:"file_ids"`
}

type ScheduleDeletionResponse struct {
	BatchID      string `json:"batch_id"`
	FileCount    int    `json:"file_count"`
	ScheduledFor string `json:"scheduled_for"`
	Status       string `json:"status"`
}

type DeletionAuditItem struct {
	AuditID       string `json:"audit_id"`
	FileID        string `json:"file_id,omitempty"`
	CampaignID    string `json:"campaign_id,omitempty"`
	Action        string `json:"action"`
	TriggeredBy   string `json:"triggered_by,omitempty"`
	FileSizeBytes int64  `json:"file_size_bytes,omitempty"`
	Reason        string `json:"reason,omitempty"`
	InitiatedAt   string `json:"initiated_at"`
	CompletedAt   string `json:"completed_at"`
}

type DeletionAuditQueryResponse struct {
	Deletions         []DeletionAuditItem `json:"deletions"`
	TotalFilesDeleted int64               `json:"total_files_deleted"`
	TotalSizeFreed    int64               `json:"total_size_freed"`
}

type CacheMetricsResponse struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}
