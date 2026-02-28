package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status    string       `json:"status"`
	Code      string       `json:"code,omitempty"`
	Message   string       `json:"message,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
	Error     ErrorPayload `json:"error"`
}

type CreatePolicyRequest struct {
	DataType            string              `json:"data_type"`
	RetentionYears      int                 `json:"retention_years"`
	SoftDeleteGraceDays int                 `json:"soft_delete_grace_days,omitempty"`
	SelectiveRules      map[string][]string `json:"selective_retention_rules,omitempty"`
}

type CreatePreviewRequest struct {
	PolicyID string `json:"policy_id,omitempty"`
	DataType string `json:"data_type,omitempty"`
}

type ApprovePreviewRequest struct {
	Reason string `json:"reason"`
}

type CreateLegalHoldRequest struct {
	EntityID  string `json:"entity_id"`
	DataType  string `json:"data_type"`
	Reason    string `json:"reason"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type CreateRestorationRequest struct {
	EntityID        string `json:"entity_id"`
	DataType        string `json:"data_type"`
	Reason          string `json:"reason"`
	ArchiveLocation string `json:"archive_location,omitempty"`
}

type ApproveRestorationRequest struct {
	Reason string `json:"reason"`
}

type RetentionPolicyResponse struct {
	PolicyID            string              `json:"policy_id"`
	DataType            string              `json:"data_type"`
	RetentionYears      int                 `json:"retention_years"`
	SoftDeleteGraceDays int                 `json:"soft_delete_grace_days"`
	SelectiveRules      map[string][]string `json:"selective_retention_rules,omitempty"`
	Status              string              `json:"status"`
	CreatedBy           string              `json:"created_by"`
	CreatedAt           string              `json:"created_at"`
}

type PoliciesResponse struct {
	Items []RetentionPolicyResponse `json:"items"`
}

type DeletionPreviewResponse struct {
	PreviewID            string `json:"preview_id"`
	PolicyID             string `json:"policy_id,omitempty"`
	DataType             string `json:"data_type"`
	TotalRecordsToDelete int    `json:"total_records_to_delete"`
	EstimatedBytes       int64  `json:"estimated_bytes"`
	WillBeArchivedTo     string `json:"will_be_archived_to"`
	Status               string `json:"status"`
	RequestedBy          string `json:"requested_by"`
	CreatedAt            string `json:"created_at"`
	ApprovedAt           string `json:"approved_at,omitempty"`
}

type ScheduledDeletionResponse struct {
	DeletionID   string `json:"deletion_id"`
	PreviewID    string `json:"preview_id"`
	PolicyID     string `json:"policy_id,omitempty"`
	DataType     string `json:"data_type"`
	Status       string `json:"status"`
	RecordsCount int    `json:"records_count"`
	Reason       string `json:"reason"`
	ScheduledAt  string `json:"scheduled_at"`
}

type LegalHoldResponse struct {
	HoldID    string `json:"hold_id"`
	EntityID  string `json:"entity_id"`
	DataType  string `json:"data_type"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	IssuedBy  string `json:"issued_by"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type LegalHoldsResponse struct {
	Items []LegalHoldResponse `json:"items"`
}

type RestorationResponse struct {
	RestorationID   string `json:"restoration_id"`
	EntityID        string `json:"entity_id"`
	DataType        string `json:"data_type"`
	Reason          string `json:"reason"`
	ArchiveLocation string `json:"archive_location,omitempty"`
	Status          string `json:"status"`
	RequestedBy     string `json:"requested_by"`
	CreatedAt       string `json:"created_at"`
	ApprovedAt      string `json:"approved_at,omitempty"`
}

type ComplianceReportResponse struct {
	PolicyCount           int `json:"policy_count"`
	ActiveLegalHolds      int `json:"active_legal_holds"`
	PendingDeletions      int `json:"pending_deletions"`
	TotalScheduledRecords int `json:"total_scheduled_records"`
	RestorationRequests   int `json:"restoration_requests"`
}
