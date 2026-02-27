package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventTeamCreated        = "team.created"
	EventTeamMemberAdded    = "team.member.added"
	EventTeamMemberRemoved  = "team.member.removed"
	EventTeamInviteSent     = "team.invite.sent"
	EventTeamInviteAccepted = "team.invite.accepted"
	EventTeamRoleChanged    = "team.role.changed"
)

func IsCanonicalInputEvent(string) bool { return false }

func IsCanonicalEmittedEvent(eventType string) bool {
	switch eventType {
	case EventTeamCreated, EventTeamMemberAdded, EventTeamMemberRemoved, EventTeamInviteSent, EventTeamInviteAccepted, EventTeamRoleChanged:
		return true
	default:
		return false
	}
}

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventTeamCreated, EventTeamMemberAdded, EventTeamMemberRemoved, EventTeamInviteSent, EventTeamInviteAccepted, EventTeamRoleChanged:
		return CanonicalEventClassDomain
	default:
		return ""
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	switch eventType {
	case EventTeamCreated, EventTeamMemberAdded, EventTeamMemberRemoved, EventTeamInviteSent, EventTeamInviteAccepted, EventTeamRoleChanged:
		return "data.team_id"
	default:
		return ""
	}
}
