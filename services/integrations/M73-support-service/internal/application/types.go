package application

import "time"

type Config struct {
	ServiceName    string
	Version        string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateTicketInput struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority,omitempty"`
	Channel     string `json:"channel,omitempty"`
	EntityType  string `json:"entity_type,omitempty"`
	EntityID    string `json:"entity_id,omitempty"`
}

type UpdateTicketInput struct {
	Status    string `json:"status,omitempty"`
	SubStatus string `json:"sub_status,omitempty"`
	Priority  string `json:"priority,omitempty"`
}

type CreateFromEmailInput struct {
	SenderEmail string `json:"sender_email"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

type AssignTicketInput struct {
	AgentID string `json:"agent_id"`
	Reason  string `json:"reason,omitempty"`
}

type AddReplyInput struct {
	ReplyType string `json:"reply_type"`
	Body      string `json:"body"`
}

type SubmitCSATInput struct {
	Rating          int    `json:"rating"`
	FeedbackComment string `json:"feedback_comment,omitempty"`
}

type SearchTicketsInput struct {
	Query      string
	Status     string
	Category   string
	UserID     string
	AssignedTo string
	Limit      int
}
