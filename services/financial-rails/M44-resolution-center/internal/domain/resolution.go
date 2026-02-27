package domain

import (
	"strings"
	"time"
)

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventSubmissionApproved = "submission.approved"
	EventPayoutFailed       = "payout.failed"
	EventDisputeCreated     = "dispute.created"
	EventDisputeResolved    = "dispute.resolved"
)

const (
	DisputeTypeRefundRequest = "refund_request"
	DisputeTypeChargeback    = "chargeback"
	DisputeTypeComplaint     = "complaint"
)

const (
	DisputeStatusSubmitted      = "submitted"
	DisputeStatusUnderReview    = "under_review"
	DisputeStatusEscalated      = "escalated"
	DisputeStatusAwaitingAction = "awaiting_action"
	DisputeStatusResolved       = "resolved"
	DisputeStatusWithdrawn      = "withdrawn"
)

const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

const (
	ResolutionTypeRefundIssued     = "refund_issued"
	ResolutionTypePartialRefund    = "partial_refund"
	ResolutionTypeRefundDenied     = "refund_denied"
	ResolutionTypeNoActionRequired = "no_action_required"
)

type EvidenceFile struct {
	Filename string `json:"filename"`
	FileURL  string `json:"file_url"`
}

type Dispute struct {
	DisputeID            string         `json:"dispute_id"`
	DisputeType          string         `json:"dispute_type"`
	Status               string         `json:"status"`
	Priority             string         `json:"priority"`
	UserID               string         `json:"user_id"`
	RecipientID          string         `json:"recipient_id,omitempty"`
	TransactionID        string         `json:"transaction_id"`
	EntityType           string         `json:"entity_type"`
	EntityID             string         `json:"entity_id"`
	ReasonCategory       string         `json:"reason_category"`
	JustificationText    string         `json:"justification_text"`
	RequestedAmount      float64        `json:"requested_amount"`
	ApprovedRefundAmount float64        `json:"approved_refund_amount,omitempty"`
	ResolutionType       string         `json:"resolution_type,omitempty"`
	ResolutionNotes      string         `json:"resolution_notes,omitempty"`
	AssignedAgentID      string         `json:"assigned_agent_id,omitempty"`
	ExpectedResolution   *time.Time     `json:"expected_resolution,omitempty"`
	SLAHoursTarget       int            `json:"sla_hours_target"`
	SLABreached          bool           `json:"sla_breached"`
	RefundPending        bool           `json:"refund_pending"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	ResolvedAt           *time.Time     `json:"resolved_at,omitempty"`
	EvidenceFiles        []EvidenceFile `json:"evidence_files,omitempty"`
}

type DisputeMessage struct {
	MessageID   string         `json:"message_id"`
	DisputeID   string         `json:"dispute_id"`
	SenderID    string         `json:"sender_id"`
	MessageBody string         `json:"message_body"`
	Attachments []EvidenceFile `json:"attachments,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type DisputeApproval struct {
	ApprovalID      string    `json:"approval_id"`
	DisputeID       string    `json:"dispute_id"`
	ApprovedBy      string    `json:"approved_by"`
	ApprovalLevel   string    `json:"approval_level"`
	RefundAmount    float64   `json:"refund_amount"`
	ApprovalReason  string    `json:"approval_reason"`
	ResolutionNotes string    `json:"resolution_notes"`
	Status          string    `json:"status"`
	ApprovedAt      time.Time `json:"approved_at"`
}

type DisputeStateHistory struct {
	HistoryID  string    `json:"history_id"`
	DisputeID  string    `json:"dispute_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ChangedBy  string    `json:"changed_by"`
	Reason     string    `json:"reason"`
	ChangedAt  time.Time `json:"changed_at"`
}

type DisputeAuditLog struct {
	AuditLogID string            `json:"audit_log_id"`
	DisputeID  string            `json:"dispute_id,omitempty"`
	ActionType string            `json:"action_type"`
	ActorID    string            `json:"actor_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

type DisputeEvidence struct {
	EvidenceID       string    `json:"evidence_id"`
	DisputeID        string    `json:"dispute_id"`
	UploadedByUserID string    `json:"uploaded_by_user_id"`
	FileURL          string    `json:"file_url"`
	Filename         string    `json:"filename"`
	FileSize         int64     `json:"file_size"`
	MimeType         string    `json:"mime_type"`
	UploadedAt       time.Time `json:"uploaded_at"`
	Scanned          bool      `json:"scanned"`
	ScanResult       string    `json:"scan_result"`
}

type DisputeMediation struct {
	MediationID string    `json:"mediation_id"`
	DisputeID   string    `json:"dispute_id"`
	MediatorID  string    `json:"mediator_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type AutoResolutionRule struct {
	RuleID    string    `json:"rule_id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DisputeDetail struct {
	Dispute      Dispute               `json:"dispute"`
	Messages     []DisputeMessage      `json:"messages,omitempty"`
	StateHistory []DisputeStateHistory `json:"state_history,omitempty"`
}

func NormalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "user":
		return "user"
	case "agent":
		return "agent"
	case "manager":
		return "manager"
	case "director":
		return "director"
	case "legal":
		return "legal"
	case "admin":
		return "admin"
	default:
		return ""
	}
}

func NormalizeDisputeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case DisputeTypeRefundRequest:
		return DisputeTypeRefundRequest
	case DisputeTypeChargeback:
		return DisputeTypeChargeback
	case DisputeTypeComplaint:
		return DisputeTypeComplaint
	default:
		return ""
	}
}

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventSubmissionApproved, EventPayoutFailed, EventDisputeCreated:
		return CanonicalEventClassDomain
	case EventDisputeResolved:
		return CanonicalEventClassAnalyticsOnly
	default:
		return ""
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	switch eventType {
	case EventSubmissionApproved:
		return "data.submission_id"
	case EventPayoutFailed:
		return "data.payout_id"
	case EventDisputeCreated, EventDisputeResolved:
		return "data.dispute_id"
	default:
		return ""
	}
}

func IsCanonicalInputEvent(eventType string) bool {
	switch eventType {
	case EventSubmissionApproved, EventPayoutFailed:
		return true
	default:
		return false
	}
}

func ValidateJustification(text string) error {
	l := len(strings.TrimSpace(text))
	if l < 50 || l > 1000 {
		return ErrInvalidInput
	}
	return nil
}

func ValidateStatusTransition(from, to string) error {
	if from == to {
		return nil
	}
	allowed := map[string]map[string]bool{
		DisputeStatusSubmitted:      {DisputeStatusUnderReview: true, DisputeStatusWithdrawn: true, DisputeStatusEscalated: true},
		DisputeStatusUnderReview:    {DisputeStatusAwaitingAction: true, DisputeStatusResolved: true, DisputeStatusEscalated: true, DisputeStatusMediation(): true},
		DisputeStatusEscalated:      {DisputeStatusResolved: true, DisputeStatusAwaitingAction: true},
		DisputeStatusAwaitingAction: {DisputeStatusResolved: true},
	}
	if next, ok := allowed[from]; ok && next[to] {
		return nil
	}
	return ErrInvalidStateTransition
}

func DisputeStatusMediation() string { return "mediation" }

func PriorityForAmount(amount float64) string {
	if amount > 1000 {
		return PriorityHigh
	}
	return PriorityNormal
}

func DefaultSLAHours(disputeType string) int {
	switch disputeType {
	case DisputeTypeChargeback:
		return 168
	case DisputeTypeComplaint:
		return 72
	default:
		return 120
	}
}

func CanApproveRefund(role string, amount float64) bool {
	r := NormalizeRole(role)
	switch r {
	case "admin", "legal":
		return true
	case "director":
		return amount <= 10000
	case "manager":
		return amount <= 1000
	case "agent":
		return amount <= 100
	default:
		return false
	}
}

func ApprovalLevelForRole(role string) string {
	switch NormalizeRole(role) {
	case "agent":
		return "agent"
	case "manager":
		return "manager"
	case "director":
		return "director"
	case "legal":
		return "legal"
	case "admin":
		return "admin"
	default:
		return "user"
	}
}
