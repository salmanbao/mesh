package domain

import "time"

type Ticket struct {
	TicketID         string    `json:"ticket_id"`
	UserID           string    `json:"user_id"`
	Subject          string    `json:"subject"`
	Description      string    `json:"description"`
	Category         string    `json:"category"`
	Priority         string    `json:"priority"`
	Status           string    `json:"status"`
	SubStatus        string    `json:"sub_status"`
	Channel          string    `json:"channel"`
	EntityType       string    `json:"entity_type,omitempty"`
	EntityID         string    `json:"entity_id,omitempty"`
	AssignedAgentID  string    `json:"assigned_agent_id,omitempty"`
	SLAResponseDueAt time.Time `json:"sla_response_due_at"`
	FirstResponseAt  time.Time `json:"first_response_at,omitempty"`
	ClosedAt         time.Time `json:"closed_at,omitempty"`
	DeletedAt        time.Time `json:"deleted_at,omitempty"`
	LastActivityAt   time.Time `json:"last_activity_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TicketReply struct {
	ReplyID   string    `json:"reply_id"`
	TicketID  string    `json:"ticket_id"`
	AuthorID  string    `json:"author_id"`
	ReplyType string    `json:"reply_type"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type CSATRating struct {
	CSATID          string    `json:"csat_id"`
	TicketID        string    `json:"ticket_id"`
	UserID          string    `json:"user_id"`
	Rating          int       `json:"rating"`
	FeedbackComment string    `json:"feedback_comment,omitempty"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

type Agent struct {
	AgentID         string   `json:"agent_id"`
	Role            string   `json:"role"`
	SkillTags       []string `json:"skill_tags"`
	OpenTicketCount int      `json:"open_ticket_count"`
	Active          bool     `json:"active"`
}

type SearchFilter struct {
	Query      string
	Status     string
	Category   string
	UserID     string
	AssignedTo string
	Limit      int
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Response    []byte
	ExpiresAt   time.Time
}
