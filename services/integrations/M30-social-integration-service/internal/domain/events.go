package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventSocialAccountConnected    = "social.account.connected"
	EventSocialPostValidated       = "social.post.validated"
	EventSocialComplianceViolation = "social.compliance.violation"
	EventSocialStatusChanged       = "social.status_changed"
	EventSocialFollowersSynced     = "social.followers_synced"
)

var canonicalInputEvents = map[string]string{
	EventSocialAccountConnected:    CanonicalEventClassDomain,
	EventSocialPostValidated:       CanonicalEventClassDomain,
	EventSocialComplianceViolation: CanonicalEventClassDomain,
	EventSocialStatusChanged:       CanonicalEventClassDomain,
	EventSocialFollowersSynced:     CanonicalEventClassDomain,
}

func IsCanonicalInputEvent(eventType string) bool {
	_, ok := canonicalInputEvents[eventType]
	return ok
}

func IsCanonicalEmittedEvent(string) bool { return false }

func CanonicalEventClass(eventType string) string {
	if class, ok := canonicalInputEvents[eventType]; ok {
		return class
	}
	return ""
}

func CanonicalPartitionKeyPath(eventType string) string {
	if IsCanonicalInputEvent(eventType) {
		return "data.user_id"
	}
	return ""
}
