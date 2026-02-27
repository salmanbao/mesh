package domain

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	EventTransactionSucceeded = "transaction.succeeded"
	EventTransactionFailed    = "transaction.failed"
	EventTransactionRefunded  = "transaction.refunded"
)
