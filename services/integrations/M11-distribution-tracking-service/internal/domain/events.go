package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const EventTrackingMetricsUpdated = "tracking.metrics.updated"

func IsCanonicalInputEvent(string) bool { return false }

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventTrackingMetricsUpdated:
		return CanonicalEventClassDomain
	default:
		return ""
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	switch eventType {
	case EventTrackingMetricsUpdated:
		return "data.tracked_post_id"
	default:
		return ""
	}
}
