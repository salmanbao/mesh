package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/ports"
)

type votingClient struct{ endpoint string }
type socialClient struct{ endpoint string }
type trackingClient struct{ endpoint string }
type submissionClient struct{ endpoint string }
type financeClient struct{ endpoint string }

func NewVotingClient(endpoint string) ports.VotingReader { return &votingClient{endpoint: endpoint} }
func NewSocialClient(endpoint string) ports.SocialReader { return &socialClient{endpoint: endpoint} }
func NewTrackingClient(endpoint string) ports.TrackingReader {
	return &trackingClient{endpoint: endpoint}
}
func NewSubmissionClient(endpoint string) ports.SubmissionReader {
	return &submissionClient{endpoint: endpoint}
}
func NewFinanceClient(endpoint string) ports.FinanceReader { return &financeClient{endpoint: endpoint} }

func failForEndpoint(endpoint string) error {
	if strings.Contains(strings.ToLower(endpoint), "fail") {
		return errors.New("upstream unavailable")
	}
	return nil
}

func (c *votingClient) GetVoteSummary(_ context.Context, _ string, _ time.Time, _ time.Time) (ports.VoteSummary, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.VoteSummary{}, err
	}
	return ports.VoteSummary{TotalVotes: 1200}, nil
}

func (c *socialClient) GetSocialSummary(_ context.Context, _ string) (ports.SocialSummary, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.SocialSummary{}, err
	}
	return ports.SocialSummary{LinkedAccounts: 3}, nil
}

func (c *trackingClient) GetTrackingSummary(_ context.Context, _ string, _ time.Time, _ time.Time) (ports.TrackingSummary, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.TrackingSummary{}, err
	}
	return ports.TrackingSummary{TotalViews: 150000}, nil
}

func (c *submissionClient) GetSubmissionSummary(_ context.Context, _ string, _ time.Time, _ time.Time) (ports.SubmissionSummary, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.SubmissionSummary{}, err
	}
	return ports.SubmissionSummary{TotalSubmissions: 42, Approved: 35}, nil
}

func (c *financeClient) GetFinanceSummary(_ context.Context, _ string, _ time.Time, _ time.Time) (ports.FinanceSummary, error) {
	if err := failForEndpoint(c.endpoint); err != nil {
		return ports.FinanceSummary{}, err
	}
	return ports.FinanceSummary{GMV: 2300.5, Refunds: 120.0, Payouts: 980.0}, nil
}
