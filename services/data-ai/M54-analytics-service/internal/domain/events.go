package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventSubmissionCreated      = "submission.created"
	EventSubmissionApproved     = "submission.approved"
	EventPayoutPaid             = "payout.paid"
	EventRewardCalculated       = "reward.calculated"
	EventCampaignLaunched       = "campaign.launched"
	EventUserRegistered         = "user.registered"
	EventTransactionSucceeded   = "transaction.succeeded"
	EventTransactionRefunded    = "transaction.refunded"
	EventTrackingMetricsUpdated = "tracking.metrics.updated"
	EventDiscoverItemClicked    = "discover.item_clicked"
	EventDeliveryDownloadDone   = "delivery.download_completed"
	EventConsentUpdated         = "consent.updated"
)

var canonicalEvents = map[string]struct{}{
	EventSubmissionCreated:      {},
	EventSubmissionApproved:     {},
	EventPayoutPaid:             {},
	EventRewardCalculated:       {},
	EventCampaignLaunched:       {},
	EventUserRegistered:         {},
	EventTransactionSucceeded:   {},
	EventTransactionRefunded:    {},
	EventTrackingMetricsUpdated: {},
	EventDiscoverItemClicked:    {},
	EventDeliveryDownloadDone:   {},
	EventConsentUpdated:         {},
}

func IsCanonicalAnalyticsInputEvent(eventType string) bool {
	_, ok := canonicalEvents[eventType]
	return ok
}

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventDiscoverItemClicked:
		return CanonicalEventClassAnalyticsOnly
	default:
		return CanonicalEventClassDomain
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	switch eventType {
	case EventSubmissionCreated, EventSubmissionApproved, EventRewardCalculated:
		return "data.submission_id"
	case EventPayoutPaid:
		return "data.payout_id"
	case EventCampaignLaunched:
		return "data.campaign_id"
	case EventUserRegistered, EventConsentUpdated:
		return "data.user_id"
	case EventTransactionSucceeded, EventTransactionRefunded:
		return "data.transaction_id"
	case EventTrackingMetricsUpdated:
		return "data.tracked_post_id"
	case EventDiscoverItemClicked:
		return "data.session_id"
	case EventDeliveryDownloadDone:
		return "data.download_id"
	default:
		return ""
	}
}
