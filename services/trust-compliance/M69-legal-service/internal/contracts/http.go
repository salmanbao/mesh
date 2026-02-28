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

type UploadDocumentRequest struct {
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
}

type RequestSignatureRequest struct {
	SignerUserID string `json:"signer_user_id"`
}

type CreateHoldRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Reason     string `json:"reason"`
}

type ReleaseHoldRequest struct {
	Reason string `json:"reason"`
}

type ComplianceScanRequest struct {
	ReportType string `json:"report_type,omitempty"`
}

type CreateDisputeRequest struct {
	UserID        string `json:"user_id"`
	OpposingParty string `json:"opposing_party"`
	DisputeReason string `json:"dispute_reason"`
	AmountCents   int64  `json:"amount_cents,omitempty"`
}

type CreateDMCANoticeRequest struct {
	ContentID string `json:"content_id"`
	Claimant  string `json:"claimant"`
	Reason    string `json:"reason"`
}

type GenerateFilingRequest struct {
	UserID  string `json:"user_id"`
	TaxYear int    `json:"tax_year"`
}

type LegalDocumentResponse struct {
	DocumentID   string `json:"document_id"`
	DocumentType string `json:"document_type"`
	FileName     string `json:"file_name"`
	Status       string `json:"status"`
	UploadedBy   string `json:"uploaded_by"`
	CreatedAt    string `json:"created_at"`
}

type SignatureResponse struct {
	SignatureID  string `json:"signature_id"`
	DocumentID   string `json:"document_id"`
	SignerUserID string `json:"signer_user_id"`
	Status       string `json:"status"`
	RequestedBy  string `json:"requested_by"`
	RequestedAt  string `json:"requested_at"`
}

type HoldResponse struct {
	HoldID     string `json:"hold_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
	IssuedBy   string `json:"issued_by"`
	CreatedAt  string `json:"created_at"`
	ReleasedAt string `json:"released_at,omitempty"`
}

type HoldCheckResponse struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Held       bool   `json:"held"`
	HoldID     string `json:"hold_id,omitempty"`
}

type ComplianceReportResponse struct {
	ReportID      string `json:"report_id"`
	ReportType    string `json:"report_type"`
	Status        string `json:"status"`
	FindingsCount int    `json:"findings_count"`
	DownloadURL   string `json:"download_url"`
	CreatedBy     string `json:"created_by"`
	CreatedAt     string `json:"created_at"`
}

type DisputeResponse struct {
	DisputeID     string `json:"dispute_id"`
	UserID        string `json:"user_id"`
	OpposingParty string `json:"opposing_party"`
	DisputeReason string `json:"dispute_reason"`
	AmountCents   int64  `json:"amount_cents"`
	Status        string `json:"status"`
	EvidenceCount int    `json:"evidence_count"`
	CreatedAt     string `json:"created_at"`
}

type DMCANoticeResponse struct {
	NoticeID   string `json:"notice_id"`
	ContentID  string `json:"content_id"`
	Claimant   string `json:"claimant"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
	ReceivedAt string `json:"received_at"`
}

type FilingResponse struct {
	FilingID      string `json:"filing_id"`
	FilingType    string `json:"filing_type"`
	TaxYear       int    `json:"tax_year"`
	UserID        string `json:"user_id"`
	Status        string `json:"status"`
	TaxDocumentID string `json:"tax_document_id"`
	CreatedAt     string `json:"created_at"`
}
