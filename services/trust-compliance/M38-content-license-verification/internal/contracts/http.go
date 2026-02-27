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

type ScanLicenseRequest struct {
	SubmissionID      string `json:"submission_id"`
	CreatorID         string `json:"creator_id"`
	MediaType         string `json:"media_type"`
	MediaURL          string `json:"media_url"`
	DeclaredLicenseID string `json:"declared_license_id,omitempty"`
}

type ScanLicenseResponse struct {
	MatchID         string  `json:"match_id"`
	SubmissionID    string  `json:"submission_id"`
	ConfidenceScore float64 `json:"confidence_score"`
	Decision        string  `json:"decision"`
	HoldID          string  `json:"hold_id,omitempty"`
	ScannedAt       string  `json:"scanned_at"`
}

type FileAppealRequest struct {
	SubmissionID       string `json:"submission_id"`
	HoldID             string `json:"hold_id,omitempty"`
	CreatorID          string `json:"creator_id"`
	CreatorExplanation string `json:"creator_explanation"`
}

type FileAppealResponse struct {
	AppealID        string `json:"appeal_id"`
	SubmissionID    string `json:"submission_id"`
	HoldID          string `json:"hold_id"`
	Status          string `json:"status"`
	AppealCreatedAt string `json:"appeal_created_at"`
}

type DMCATakedownRequest struct {
	SubmissionID     string `json:"submission_id"`
	RightsHolderName string `json:"rights_holder_name"`
	ContactEmail     string `json:"contact_email"`
	Reference        string `json:"reference"`
}

type DMCATakedownResponse struct {
	DMCAID           string `json:"dmca_id"`
	SubmissionID     string `json:"submission_id"`
	Status           string `json:"status"`
	NoticeReceivedAt string `json:"notice_received_at"`
}
