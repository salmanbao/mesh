package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventEscrowHoldCreated       = "escrow.hold_created"
	EventEscrowHoldFullyReleased = "escrow.hold_fully_released"
	EventEscrowPartialRelease    = "escrow.partial_release"
	EventEscrowRefundProcessed   = "escrow.refund_processed"
)

func IsCanonicalInputEvent(string) bool { return false }

func IsCanonicalEmittedEvent(eventType string) bool {
	switch eventType {
	case EventEscrowHoldCreated, EventEscrowHoldFullyReleased, EventEscrowPartialRelease, EventEscrowRefundProcessed:
		return true
	default:
		return false
	}
}

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventEscrowRefundProcessed:
		return CanonicalEventClassDomain
	case EventEscrowHoldCreated, EventEscrowHoldFullyReleased, EventEscrowPartialRelease:
		return CanonicalEventClassAnalyticsOnly
	default:
		return ""
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	if IsCanonicalEmittedEvent(eventType) {
		return "data.escrow_id"
	}
	return ""
}
