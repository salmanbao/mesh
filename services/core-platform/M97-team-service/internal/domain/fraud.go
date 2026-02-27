package domain

import "time"

const (
	TeamStatusActive = "active"

	MemberStatusActive  = "active"
	MemberStatusRemoved = "removed"

	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusExpired  = "expired"
)

type Team struct {
	TeamID    string
	ScopeType string
	ScopeID   string
	OwnerID   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TeamMember struct {
	TeamMemberID string
	TeamID       string
	UserID       string
	Role         string
	Status       string
	JoinedAt     time.Time
	RemovedAt    *time.Time
}

type Invite struct {
	InviteID   string
	TeamID     string
	Email      string
	Role       string
	Status     string
	InvitedBy  string
	ExpiresAt  time.Time
	Token      string
	AcceptedBy string
	AcceptedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AuditLog struct {
	AuditID      string
	TeamID       string
	ActorUserID  string
	Action       string
	TargetUserID string
	Metadata     map[string]string
	OccurredAt   time.Time
}

type RolePolicy struct {
	Role        string
	Permissions []string
	CreatedAt   time.Time
}

type MembershipResult struct {
	Allowed bool
	Role    string
}

func IsValidScopeType(v string) bool {
	switch v {
	case "storefront", "account":
		return true
	default:
		return false
	}
}

func IsValidRole(v string) bool {
	switch v {
	case "owner", "admin", "editor", "viewer":
		return true
	default:
		return false
	}
}
