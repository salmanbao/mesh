package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
)

func (s *Service) ProcessNextJob(ctx context.Context) error {
	jobID, err := s.queue.Dequeue(ctx)
	if err != nil {
		return err
	}
	job, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return err
	}
	asset, err := s.assets.GetByID(ctx, job.AssetID)
	if err != nil {
		return err
	}

	started := s.nowFn()
	job.Status = domain.JobStatusProcessing
	job.Attempts++
	job.StartedAt = started
	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}

	if strings.ToLower(strings.TrimSpace(asset.MIMEType)) != "video/mp4" && strings.ToLower(strings.TrimSpace(asset.MIMEType)) != "video/quicktime" {
		return s.failJob(ctx, asset, job, "unsupported_codec")
	}

	switch job.JobType {
	case domain.JobTypeTranscode1080:
		err = s.outputs.Upsert(ctx, domain.MediaOutput{OutputID: uuid.NewString(), AssetID: asset.AssetID, Profile: domain.Profile1080, AspectRatio: domain.Aspect169, S3URL: fmt.Sprintf("https://cdn.viralforge/media/%s/1080p.mp4", asset.AssetID), CreatedAt: s.nowFn()})
	case domain.JobTypeTranscode720:
		err = s.outputs.Upsert(ctx, domain.MediaOutput{OutputID: uuid.NewString(), AssetID: asset.AssetID, Profile: domain.Profile720, AspectRatio: domain.Aspect169, S3URL: fmt.Sprintf("https://cdn.viralforge/media/%s/720p.mp4", asset.AssetID), CreatedAt: s.nowFn()})
	case domain.JobTypeAspect916:
		err = s.outputs.Upsert(ctx, domain.MediaOutput{OutputID: uuid.NewString(), AssetID: asset.AssetID, Profile: domain.Profile1080, AspectRatio: domain.Aspect916, S3URL: fmt.Sprintf("https://cdn.viralforge/media/%s/9x16-1080p.mp4", asset.AssetID), CreatedAt: s.nowFn()})
		if err == nil {
			err = s.outputs.Upsert(ctx, domain.MediaOutput{OutputID: uuid.NewString(), AssetID: asset.AssetID, Profile: domain.Profile720, AspectRatio: domain.Aspect916, S3URL: fmt.Sprintf("https://cdn.viralforge/media/%s/9x16-720p.mp4", asset.AssetID), CreatedAt: s.nowFn()})
		}
	case domain.JobTypeAspect11:
		err = s.outputs.Upsert(ctx, domain.MediaOutput{OutputID: uuid.NewString(), AssetID: asset.AssetID, Profile: domain.Profile1080, AspectRatio: domain.Aspect11, S3URL: fmt.Sprintf("https://cdn.viralforge/media/%s/1x1-1080p.mp4", asset.AssetID), CreatedAt: s.nowFn()})
		if err == nil {
			err = s.outputs.Upsert(ctx, domain.MediaOutput{OutputID: uuid.NewString(), AssetID: asset.AssetID, Profile: domain.Profile720, AspectRatio: domain.Aspect11, S3URL: fmt.Sprintf("https://cdn.viralforge/media/%s/1x1-720p.mp4", asset.AssetID), CreatedAt: s.nowFn()})
		}
	case domain.JobTypeThumbnails:
		for _, ratio := range []domain.AspectRatio{domain.Aspect169, domain.Aspect916, domain.Aspect11} {
			for _, pos := range []string{"10%", "50%", "90%"} {
				err = s.thumbnails.Upsert(ctx, domain.MediaThumbnail{
					ThumbnailID: uuid.NewString(),
					AssetID:     asset.AssetID,
					Position:    pos,
					AspectRatio: ratio,
					S3URL:       fmt.Sprintf("https://cdn.viralforge/thumbnails/%s/%s/%s.jpg", asset.AssetID, strings.ReplaceAll(string(ratio), ":", "x"), strings.TrimSuffix(pos, "%")),
					CreatedAt:   s.nowFn(),
				})
				if err != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
	case domain.JobTypeWatermark:
		if asset.ApprovalStatus == domain.ApprovalStatusApproved {
			err = s.watermarks.Upsert(ctx, domain.WatermarkRecord{WatermarkID: uuid.NewString(), AssetID: asset.AssetID, WatermarkText: fmt.Sprintf("wmk_%s", asset.SubmissionID), Placement: "bottom-right", Opacity: 0.25, AppliedAt: s.nowFn()})
		}
	default:
		return s.failJob(ctx, asset, job, "unsupported_job_type")
	}
	if err != nil {
		return s.failJob(ctx, asset, job, err.Error())
	}

	job.Status = domain.JobStatusCompleted
	job.FinishedAt = s.nowFn()
	job.ErrorMessage = ""
	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}
	return s.refreshAssetStatus(ctx, asset)
}

func (s *Service) failJob(ctx context.Context, asset domain.MediaAsset, job domain.MediaJob, reason string) error {
	now := s.nowFn()
	retryable := domain.IsRetryableJobFailure(reason)
	if retryable && job.Attempts < domain.MaxJobAttempts {
		job.Status = domain.JobStatusQueued
		job.ErrorMessage = reason
		job.QueuedAt = now
		job.FinishedAt = now
		if err := s.jobs.Update(ctx, job); err != nil {
			return err
		}
		if err := s.queue.Enqueue(ctx, job.JobID); err != nil {
			job.Status = domain.JobStatusFailed
			job.ErrorMessage = "queue_unavailable"
			job.FinishedAt = now
			_ = s.jobs.Update(ctx, job)
			asset.Status = domain.AssetStatusFailed
			asset.LastErrorCode = "queue_unavailable"
			asset.LastErrorMessage = err.Error()
			asset.UpdatedAt = now
			_ = s.assets.Update(ctx, asset)
			_ = s.campaign.UpdateMediaStatus(ctx, asset.SubmissionID, asset.AssetID, string(asset.Status), asset.LastErrorCode)
			return err
		}
		asset.Status = domain.AssetStatusProcessing
		asset.LastErrorCode = reason
		asset.LastErrorMessage = reason
		asset.UpdatedAt = now
		return s.assets.Update(ctx, asset)
	}

	job.Status = domain.JobStatusFailed
	job.ErrorMessage = reason
	job.FinishedAt = now
	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}
	if retryable && job.Attempts >= domain.MaxJobAttempts && s.dlq != nil {
		_ = s.dlq.Publish(ctx, contracts.QueueDLQRecord{JobID: job.JobID, AssetID: asset.AssetID, JobType: string(job.JobType), ErrorSummary: reason, RetryCount: job.Attempts, FirstSeenAt: now, LastErrorAt: now})
	}
	asset.LastErrorCode = reason
	asset.LastErrorMessage = reason
	asset.UpdatedAt = now
	if err := s.assets.Update(ctx, asset); err != nil {
		return err
	}
	return s.refreshAssetStatus(ctx, asset)
}

func (s *Service) refreshAssetStatus(ctx context.Context, asset domain.MediaAsset) error {
	jobs, err := s.jobs.ListByAsset(ctx, asset.AssetID)
	if err != nil {
		return err
	}
	hasFailed := false
	allCompleted := len(jobs) > 0
	for _, job := range jobs {
		if job.Status == domain.JobStatusFailed {
			hasFailed = true
		}
		if job.Status != domain.JobStatusCompleted {
			allCompleted = false
		}
	}
	if hasFailed {
		asset.Status = domain.AssetStatusFailed
	} else if allCompleted {
		asset.Status = domain.AssetStatusCompleted
	} else {
		asset.Status = domain.AssetStatusProcessing
	}
	asset.UpdatedAt = s.nowFn()
	if err := s.assets.Update(ctx, asset); err != nil {
		return err
	}
	if asset.Status == domain.AssetStatusCompleted || asset.Status == domain.AssetStatusFailed {
		_ = s.campaign.UpdateMediaStatus(ctx, asset.SubmissionID, asset.AssetID, string(asset.Status), asset.LastErrorCode)
	}
	return nil
}
