package domain

import "time"

type MigrationPlan struct {
	PlanID           string         `json:"plan_id"`
	ServiceName      string         `json:"service_name"`
	Environment      string         `json:"environment"`
	Version          string         `json:"version"`
	Plan             map[string]any `json:"plan"`
	Status           string         `json:"status"`
	DryRun           bool           `json:"dry_run"`
	RiskLevel        string         `json:"risk_level"`
	StagingValidated bool           `json:"staging_validated"`
	BackupRequired   bool           `json:"backup_required"`
	CreatedBy        string         `json:"created_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type MigrationRun struct {
	RunID             string    `json:"run_id"`
	PlanID            string    `json:"plan_id"`
	Status            string    `json:"status"`
	OperatorID        string    `json:"operator_id"`
	SnapshotCreated   bool      `json:"snapshot_created"`
	RollbackAvailable bool      `json:"rollback_available"`
	ValidationStatus  string    `json:"validation_status"`
	BackfillJobID     string    `json:"backfill_job_id"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       time.Time `json:"completed_at"`
}

type RegistryRecord struct {
	RegistryID  string    `json:"registry_id"`
	ServiceName string    `json:"service_name"`
	Environment string    `json:"environment"`
	Version     string    `json:"version"`
	Checksum    string    `json:"checksum"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type BackfillJob struct {
	JobID       string    `json:"job_id"`
	PlanID      string    `json:"plan_id"`
	ProgressPct int       `json:"progress_pct"`
	Status      string    `json:"status"`
	Checkpoint  string    `json:"checkpoint"`
	CreatedAt   time.Time `json:"created_at"`
}

type Metrics struct {
	PlanCount       int `json:"plan_count"`
	RunCount        int `json:"run_count"`
	SuccessfulRuns  int `json:"successful_runs"`
	FailedRuns      int `json:"failed_runs"`
	ActiveBackfills int `json:"active_backfills"`
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Response    []byte
	ExpiresAt   time.Time
}
