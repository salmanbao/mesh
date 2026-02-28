package domain

import "time"

const (
	PolicyStatusActive = "active"

	PreviewStatusPending  = "pending_approval"
	PreviewStatusApproved = "approved"

	LegalHoldStatusActive = "active"

	RestorationStatusPending  = "pending_approval"
	RestorationStatusApproved = "approved"

	ScheduledDeletionStatusScheduled = "scheduled"
)

type RetentionPolicy struct {
	PolicyID            string              `json:"policy_id"`
	DataType            string              `json:"data_type"`
	RetentionYears      int                 `json:"retention_years"`
	SoftDeleteGraceDays int                 `json:"soft_delete_grace_days"`
	SelectiveRules      map[string][]string `json:"selective_retention_rules,omitempty"`
	Status              string              `json:"status"`
	CreatedBy           string              `json:"created_by"`
	CreatedAt           time.Time           `json:"created_at"`
}

type DeletionPreview struct {
	PreviewID            string     `json:"preview_id"`
	PolicyID             string     `json:"policy_id,omitempty"`
	DataType             string     `json:"data_type"`
	TotalRecordsToDelete int        `json:"total_records_to_delete"`
	EstimatedBytes       int64      `json:"estimated_bytes"`
	WillBeArchivedTo     string     `json:"will_be_archived_to"`
	Status               string     `json:"status"`
	RequestedBy          string     `json:"requested_by"`
	CreatedAt            time.Time  `json:"created_at"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
}

type LegalHold struct {
	HoldID    string     `json:"hold_id"`
	EntityID  string     `json:"entity_id"`
	DataType  string     `json:"data_type"`
	Reason    string     `json:"reason"`
	Status    string     `json:"status"`
	IssuedBy  string     `json:"issued_by"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type RestorationRequest struct {
	RestorationID   string     `json:"restoration_id"`
	EntityID        string     `json:"entity_id"`
	DataType        string     `json:"data_type"`
	Reason          string     `json:"reason"`
	ArchiveLocation string     `json:"archive_location,omitempty"`
	Status          string     `json:"status"`
	RequestedBy     string     `json:"requested_by"`
	CreatedAt       time.Time  `json:"created_at"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
}

type ScheduledDeletion struct {
	DeletionID   string    `json:"deletion_id"`
	PreviewID    string    `json:"preview_id"`
	PolicyID     string    `json:"policy_id,omitempty"`
	DataType     string    `json:"data_type"`
	Status       string    `json:"status"`
	RecordsCount int       `json:"records_count"`
	Reason       string    `json:"reason"`
	ScheduledAt  time.Time `json:"scheduled_at"`
}

type AuditLog struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type"`
	ActorID    string            `json:"actor_id"`
	EntityID   string            `json:"entity_id"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
