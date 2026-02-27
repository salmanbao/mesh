package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventSocialAccountConnected     = "social.account.connected"
	EventSocialComplianceViolation  = "social.compliance.violation"
	EventSocialFollowersSynced      = "social.followers_synced"
	EventSocialPostValidated        = "social.post.validated"
	EventSocialStatusChanged        = "social.status_changed"
)

func IsCanonicalInputEvent(string) bool { return false }

func IsCanonicalEmittedEvent(eventType string) bool {
	switch eventType {
	case EventSocialAccountConnected, EventSocialComplianceViolation, EventSocialFollowersSynced, EventSocialPostValidated, EventSocialStatusChanged:
		return true
	default:
		return false
	}
}

func CanonicalEventClass(eventType string) string {
	if IsCanonicalEmittedEvent(eventType) {
		return CanonicalEventClassDomain
	}
	return ""
}

func CanonicalPartitionKeyPath(eventType string) string {
	if IsCanonicalEmittedEvent(eventType) {
		return "data.user_id"
	}
	return ""
}
