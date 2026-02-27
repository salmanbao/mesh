package contracts

type FileDisputeRequest struct {
	TransactionID string `json:"transaction_id"`
	DisputeType   string `json:"dispute_type"`
	Reason        string `json:"reason"`
	BuyerClaim    string `json:"buyer_claim"`
}

type SubmitDisputeEvidenceRequest struct {
	Filename    string `json:"filename"`
	Description string `json:"description"`
	FileURL     string `json:"file_url"`
	SizeBytes   int64  `json:"size_bytes"`
	MimeType    string `json:"mime_type"`
}

type ChargebackWebhookRequest struct {
	EventID          string                `json:"event_id"`
	EventType        string                `json:"event_type"`
	OccurredAt       string                `json:"occurred_at"`
	SourceService    string                `json:"source_service"`
	TraceID          string                `json:"trace_id"`
	SchemaVersion    string                `json:"schema_version"`
	PartitionKeyPath string                `json:"partition_key_path"`
	PartitionKey     string                `json:"partition_key"`
	Data             ChargebackWebhookData `json:"data"`
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
