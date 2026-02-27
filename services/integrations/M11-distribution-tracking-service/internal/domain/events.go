package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventTrackingMetricsUpdated = "tracking.metrics.updated"
	EventTrackingPostArchived   = "tracking.post.archived"
	EventDistributionPublished  = "distribution.published"
	EventDistributionFailed     = "distribution.failed"
)

type canonicalEventMeta struct {
	class            string
	partitionKeyPath string
}

var canonicalInputEvents = map[string]canonicalEventMeta{
	EventDistributionPublished: {class: CanonicalEventClassDomain, partitionKeyPath: "data.distribution_item_id"},
	EventDistributionFailed:    {class: CanonicalEventClassDomain, partitionKeyPath: "data.distribution_item_id"},
}

func IsCanonicalInputEvent(eventType string) bool {
	_, ok := canonicalInputEvents[eventType]
	return ok
}

func CanonicalEventClass(eventType string) string {
	if m, ok := lookupCanonicalMeta(eventType); ok {
		return m.class
	}
	return ""
}

func CanonicalPartitionKeyPath(eventType string) string {
	if m, ok := lookupCanonicalMeta(eventType); ok {
		return m.partitionKeyPath
	}
	switch eventType {
	case EventTrackingMetricsUpdated:
		return "data.tracked_post_id"
	case EventTrackingPostArchived:
		return "data.tracked_post_id"
	}
	return ""
}

func lookupCanonicalMeta(eventType string) (canonicalEventMeta, bool) {
	m, ok := canonicalInputEvents[eventType]
	return m, ok
}
