package ports

import (
	"context"
	"time"
)

type VoteSummary struct {
	TotalVotes int64
}

type SocialSummary struct {
	LinkedAccounts int
}

type TrackingSummary struct {
	TotalViews int64
}

type SubmissionSummary struct {
	TotalSubmissions int
	Approved         int
}

type FinanceSummary struct {
	GMV     float64
	Refunds float64
	Payouts float64
}

type VotingReader interface {
	GetVoteSummary(ctx context.Context, userID string, from, to time.Time) (VoteSummary, error)
}

type SocialReader interface {
	GetSocialSummary(ctx context.Context, userID string) (SocialSummary, error)
}

type TrackingReader interface {
	GetTrackingSummary(ctx context.Context, userID string, from, to time.Time) (TrackingSummary, error)
}

type SubmissionReader interface {
	GetSubmissionSummary(ctx context.Context, userID string, from, to time.Time) (SubmissionSummary, error)
}

type FinanceReader interface {
	GetFinanceSummary(ctx context.Context, userID string, from, to time.Time) (FinanceSummary, error)
}
