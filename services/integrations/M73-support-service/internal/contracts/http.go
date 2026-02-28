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

type CreateTicketRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority,omitempty"`
	Channel     string `json:"channel,omitempty"`
	EntityType  string `json:"entity_type,omitempty"`
	EntityID    string `json:"entity_id,omitempty"`
}

type UpdateTicketRequest struct {
	Status    string `json:"status,omitempty"`
	SubStatus string `json:"sub_status,omitempty"`
	Priority  string `json:"priority,omitempty"`
}

type CreateFromEmailRequest struct {
	SenderEmail string `json:"sender_email"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

type AssignTicketRequest struct {
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason,omitempty"`
}

type AddReplyRequest struct {
	ReplyType string `json:"reply_type"`
	Body      string `json:"body"`
}

type SubmitCSATRequest struct {
	Rating          int    `json:"rating"`
	FeedbackComment string `json:"feedback_comment,omitempty"`
}
