package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/ports"
)

type AuthClient struct{}
type CampaignClient struct{}
type VotingClient struct{}
type TrackingClient struct{}
type SubmissionClient struct{}

func NewAuthClient(_ string) *AuthClient             { return &AuthClient{} }
func NewCampaignClient(_ string) *CampaignClient     { return &CampaignClient{} }
func NewVotingClient(_ string) *VotingClient         { return &VotingClient{} }
func NewTrackingClient(_ string) *TrackingClient     { return &TrackingClient{} }
func NewSubmissionClient(_ string) *SubmissionClient { return &SubmissionClient{} }

func (c *AuthClient) GetUser(_ context.Context, userID string) (ports.UserIdentity, error) {
	return ports.UserIdentity{UserID: userID, Email: userID + "@example.com", Role: "user"}, nil
}

func (c *CampaignClient) GetRatePer1K(_ context.Context, _ string, _ string) (float64, error) {
	return 2.5, nil
}

func (c *VotingClient) GetFraudScore(_ context.Context, _ string, _ string) (float64, error) {
	return 0.0, nil
}

func (c *TrackingClient) GetLockedViews(_ context.Context, _ string) (int64, error) {
	return 1000, nil
}

func (c *SubmissionClient) ValidateSubmission(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
