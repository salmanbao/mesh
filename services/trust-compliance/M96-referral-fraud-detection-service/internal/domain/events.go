package domain

const (
	EventAffiliateClickTracked      = "affiliate.click.tracked"
	EventAffiliateAttributionCreate = "affiliate.attribution.created"
	EventTransactionSucceeded       = "transaction.succeeded"
	EventUserRegistered             = "user.registered"
)

func IsCanonicalInputEvent(eventType string) bool {
	switch eventType {
	case EventAffiliateClickTracked, EventAffiliateAttributionCreate, EventTransactionSucceeded, EventUserRegistered:
		return true
	default:
		return false
	}
}

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventAffiliateClickTracked, EventAffiliateAttributionCreate, EventTransactionSucceeded, EventUserRegistered:
		return CanonicalEventClassDomain
	default:
		return ""
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	switch eventType {
	case EventAffiliateClickTracked, EventAffiliateAttributionCreate:
		return "data.affiliate_id"
	case EventTransactionSucceeded:
		return "data.transaction_id"
	case EventUserRegistered:
		return "data.user_id"
	default:
		return ""
	}
}
