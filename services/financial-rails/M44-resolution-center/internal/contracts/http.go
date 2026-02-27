package contracts

type EvidenceFile struct {
	Filename string `json:"filename"`
	FileURL  string `json:"file_url"`
}

type CreateDisputeRequest struct {
	DisputeType       string         `json:"dispute_type"`
	TransactionID     string         `json:"transaction_id"`
	ReasonCategory    string         `json:"reason_category"`
	JustificationText string         `json:"justification_text"`
	RequestedAmount   float64        `json:"requested_amount"`
	EvidenceFiles     []EvidenceFile `json:"evidence_files"`
}

type SendMessageRequest struct {
	MessageBody string         `json:"message_body"`
	Attachments []EvidenceFile `json:"attachments"`
}

type ApproveDisputeRequest struct {
	RefundAmount    float64 `json:"refund_amount"`
	ApprovalReason  string  `json:"approval_reason"`
	ResolutionNotes string  `json:"resolution_notes"`
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
