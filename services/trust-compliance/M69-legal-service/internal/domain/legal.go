package domain

import "time"

const (
	DocumentStatusUploaded = "uploaded"

	SignatureStatusRequested = "requested"

	HoldStatusActive   = "active"
	HoldStatusReleased = "released"

	ComplianceStatusCompleted = "completed"

	DisputeStatusOpen = "open"

	DMCANoticeStatusReceived = "received"

	FilingStatusPending = "pending"
)

type LegalDocument struct {
	DocumentID   string    `json:"document_id"`
	DocumentType string    `json:"document_type"`
	FileName     string    `json:"file_name"`
	Status       string    `json:"status"`
	UploadedBy   string    `json:"uploaded_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type SignatureRequest struct {
	SignatureID  string    `json:"signature_id"`
	DocumentID   string    `json:"document_id"`
	SignerUserID string    `json:"signer_user_id"`
	Status       string    `json:"status"`
	RequestedBy  string    `json:"requested_by"`
	RequestedAt  time.Time `json:"requested_at"`
}

type LegalHold struct {
	HoldID     string     `json:"hold_id"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	IssuedBy   string     `json:"issued_by"`
	CreatedAt  time.Time  `json:"created_at"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`
}

type ComplianceReport struct {
	ReportID      string    `json:"report_id"`
	ReportType    string    `json:"report_type"`
	Status        string    `json:"status"`
	FindingsCount int       `json:"findings_count"`
	DownloadURL   string    `json:"download_url"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type ComplianceFinding struct {
	FindingID  string    `json:"finding_id"`
	ReportID   string    `json:"report_id"`
	Regulation string    `json:"regulation"`
	Severity   string    `json:"severity"`
	Status     string    `json:"status"`
	Summary    string    `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
}

type Dispute struct {
	DisputeID     string    `json:"dispute_id"`
	UserID        string    `json:"user_id"`
	OpposingParty string    `json:"opposing_party"`
	DisputeReason string    `json:"dispute_reason"`
	AmountCents   int64     `json:"amount_cents"`
	Status        string    `json:"status"`
	EvidenceCount int       `json:"evidence_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type DMCANotice struct {
	NoticeID   string    `json:"notice_id"`
	ContentID  string    `json:"content_id"`
	Claimant   string    `json:"claimant"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	ReceivedAt time.Time `json:"received_at"`
}

type RegulatoryFiling struct {
	FilingID      string    `json:"filing_id"`
	FilingType    string    `json:"filing_type"`
	TaxYear       int       `json:"tax_year"`
	UserID        string    `json:"user_id"`
	Status        string    `json:"status"`
	TaxDocumentID string    `json:"tax_document_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditLog struct {
	AuditID    string            `json:"audit_id"`
	EventType  string            `json:"event_type"`
	ActorID    string            `json:"actor_id"`
	EntityID   string            `json:"entity_id"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
