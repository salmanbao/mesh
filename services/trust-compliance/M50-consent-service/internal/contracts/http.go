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

type UpdateConsentRequest struct {
	Preferences map[string]bool `json:"preferences"`
	Reason      string          `json:"reason"`
}

type WithdrawConsentRequest struct {
	Category string `json:"category,omitempty"`
	Reason   string `json:"reason"`
}

type ConsentRecordResponse struct {
	UserID      string          `json:"user_id"`
	Preferences map[string]bool `json:"preferences,omitempty"`
	Status      string          `json:"status"`
	UpdatedAt   string          `json:"updated_at"`
	UpdatedBy   string          `json:"updated_by"`
}

type ConsentHistoryEntryResponse struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	UserID     string `json:"user_id"`
	Category   string `json:"category,omitempty"`
	Reason     string `json:"reason"`
	ChangedBy  string `json:"changed_by"`
	OccurredAt string `json:"occurred_at"`
}

type ConsentHistoryResponse struct {
	Items []ConsentHistoryEntryResponse `json:"items"`
}
