package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

func IsCanonicalInputEvent(string) bool       { return false }
func IsCanonicalEmittedEvent(string) bool     { return false }
func CanonicalEventClass(string) string       { return "" }
func CanonicalPartitionKeyPath(string) string { return "" }
