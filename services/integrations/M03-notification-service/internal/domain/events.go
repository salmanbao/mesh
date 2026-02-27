package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventAuth2FARequired       = "auth.2fa.required"
	EventCampaignBudgetUpdated = "campaign.budget_updated"
	EventCampaignCreated       = "campaign.created"
	EventCampaignLaunched      = "campaign.launched"
	EventDisputeCreated        = "dispute.created"
	EventPayoutFailed          = "payout.failed"
	EventPayoutPaid            = "payout.paid"
	EventSubmissionApproved    = "submission.approved"
	EventSubmissionRejected    = "submission.rejected"
	EventTransactionFailed     = "transaction.failed"
	EventUserRegistered        = "user.registered"
)

type canonicalEventMeta struct {
	class            string
	partitionKeyPath string
}

var canonicalInputEvents = map[string]canonicalEventMeta{
	EventAuth2FARequired:       {class: CanonicalEventClassDomain, partitionKeyPath: "data.user_id"},
	EventCampaignBudgetUpdated: {class: CanonicalEventClassDomain, partitionKeyPath: "data.campaign_id"},
	EventCampaignCreated:       {class: CanonicalEventClassDomain, partitionKeyPath: "data.campaign_id"},
	EventCampaignLaunched:      {class: CanonicalEventClassDomain, partitionKeyPath: "data.campaign_id"},
	EventDisputeCreated:        {class: CanonicalEventClassDomain, partitionKeyPath: "data.dispute_id"},
	EventPayoutFailed:          {class: CanonicalEventClassDomain, partitionKeyPath: "data.payout_id"},
	EventPayoutPaid:            {class: CanonicalEventClassDomain, partitionKeyPath: "data.payout_id"},
	EventSubmissionApproved:    {class: CanonicalEventClassDomain, partitionKeyPath: "data.submission_id"},
	EventSubmissionRejected:    {class: CanonicalEventClassDomain, partitionKeyPath: "data.submission_id"},
	EventTransactionFailed:     {class: CanonicalEventClassDomain, partitionKeyPath: "data.transaction_id"},
	EventUserRegistered:        {class: CanonicalEventClassDomain, partitionKeyPath: "data.user_id"},
}

func IsCanonicalInputEvent(eventType string) bool {
	_, ok := canonicalInputEvents[eventType]
	return ok
}

func CanonicalEventClass(eventType string) string {
	if m, ok := canonicalInputEvents[eventType]; ok {
		return m.class
	}
	return ""
}

func CanonicalPartitionKeyPath(eventType string) string {
	if m, ok := canonicalInputEvents[eventType]; ok {
		return m.partitionKeyPath
	}
	return ""
}
