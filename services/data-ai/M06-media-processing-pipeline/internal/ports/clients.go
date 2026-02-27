package ports

import (
	"context"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
)

type CampaignReader interface {
	ValidateSubmission(ctx context.Context, submissionID string) error
	GetApprovalStatus(ctx context.Context, submissionID string) (domain.ApprovalStatus, error)
	UpdateMediaStatus(ctx context.Context, submissionID, assetID, status, reason string) error
}
