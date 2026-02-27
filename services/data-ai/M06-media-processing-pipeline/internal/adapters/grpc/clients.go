package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/ports"
)

type campaignClient struct {
	endpoint string
}

func NewCampaignClient(endpoint string) ports.CampaignReader {
	return &campaignClient{endpoint: endpoint}
}

func (c *campaignClient) fail() error {
	if strings.Contains(strings.ToLower(c.endpoint), "fail") {
		return errors.New("campaign service unavailable")
	}
	return nil
}

func (c *campaignClient) ValidateSubmission(_ context.Context, submissionID string) error {
	if err := c.fail(); err != nil {
		return err
	}
	if strings.TrimSpace(submissionID) == "" {
		return domain.ErrInvalidInput
	}
	return nil
}

func (c *campaignClient) GetApprovalStatus(_ context.Context, _ string) (domain.ApprovalStatus, error) {
	if err := c.fail(); err != nil {
		return domain.ApprovalStatusPending, err
	}
	return domain.ApprovalStatusPending, nil
}

func (c *campaignClient) UpdateMediaStatus(_ context.Context, _, _, _, _ string) error {
	return c.fail()
}
