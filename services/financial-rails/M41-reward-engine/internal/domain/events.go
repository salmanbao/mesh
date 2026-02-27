package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventSubmissionAutoApproved = "submission.auto_approved"
	EventSubmissionCancelled    = "submission.cancelled"
	EventSubmissionVerified     = "submission.verified"
	EventSubmissionViewLocked   = "submission.view_locked"
	EventTrackingMetricsUpdated = "tracking.metrics.updated"
	EventRewardCalculated       = "reward.calculated"
	EventRewardPayoutEligible   = "reward.payout_eligible"
)
