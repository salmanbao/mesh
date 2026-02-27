package ports

import "context"

type UserIdentity struct {
	UserID string
	Email  string
	Role   string
}

type AuthReader interface {
	GetUser(ctx context.Context, userID string) (UserIdentity, error)
}

type CampaignRateReader interface {
	GetRatePer1K(ctx context.Context, campaignID, userID string) (float64, error)
}

type VotingReader interface {
	GetFraudScore(ctx context.Context, submissionID, userID string) (float64, error)
}

type TrackingReader interface {
	GetLockedViews(ctx context.Context, submissionID string) (int64, error)
}

type SubmissionReader interface {
	ValidateSubmission(ctx context.Context, submissionID, userID, campaignID string) error
}
