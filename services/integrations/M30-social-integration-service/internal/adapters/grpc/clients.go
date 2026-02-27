package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/ports"
)

type SocialVerificationClient struct{}

func NewSocialVerificationClient() *SocialVerificationClient { return &SocialVerificationClient{} }

func (c *SocialVerificationClient) ListUserAccounts(context.Context, string) ([]ports.VerificationAccount, error) {
	return []ports.VerificationAccount{}, nil
}
