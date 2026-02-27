package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventAffiliateClickTracked       = "affiliate.click.tracked"
	EventAffiliateAttributionCreated = "affiliate.attribution.created"
)

func IsCanonicalInputEvent(string) bool { return false }

func IsCanonicalEmittedEvent(eventType string) bool {
	switch eventType {
	case EventAffiliateClickTracked, EventAffiliateAttributionCreated:
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
		return "data.affiliate_id"
	}
	return ""
}
