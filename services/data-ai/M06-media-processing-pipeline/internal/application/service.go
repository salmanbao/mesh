package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/contracts"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
)

func (s *Service) CreateUpload(ctx context.Context, actor Actor, input CreateUploadInput) (UploadResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return UploadResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return UploadResult{}, domain.ErrIdempotencyRequired
	}
	candidate := domain.UploadInput{
		SubmissionID:   strings.TrimSpace(input.SubmissionID),
		FileName:       strings.TrimSpace(input.FileName),
		MIMEType:       strings.TrimSpace(input.MIMEType),
		FileSize:       input.FileSize,
		ChecksumSHA256: strings.TrimSpace(input.ChecksumSHA256),
	}
	if err := domain.ValidateUploadInput(candidate); err != nil {
		return UploadResult{}, err
	}
	checksum := candidate.ChecksumSHA256
	if checksum == "" {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d", candidate.SubmissionID, candidate.FileName, candidate.MIMEType, candidate.FileSize)))
		checksum = hex.EncodeToString(sum[:])
	}
	candidate.ChecksumSHA256 = checksum
	if err := validateUploadIdempotencyKey(actor.IdempotencyKey, candidate.SubmissionID, checksum); err != nil {
		return UploadResult{}, err
	}

	now := s.nowFn()
	requestHash := hashPayload(candidate)
	rec, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return UploadResult{}, err
	}
	if rec != nil {
		if rec.RequestHash != requestHash {
			return UploadResult{}, domain.ErrIdempotencyConflict
		}
		if len(rec.ResponseBody) == 0 {
			return UploadResult{}, domain.ErrIdempotencyInFlight
		}
		var out UploadResult
		if err := json.Unmarshal(rec.ResponseBody, &out); err != nil {
			return UploadResult{}, err
		}
		return out, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return UploadResult{}, err
	}
	completed := false
	defer func() {
		if !completed {
			_ = s.idempotency.Release(ctx, actor.IdempotencyKey)
		}
	}()

	if err := s.campaign.ValidateSubmission(ctx, candidate.SubmissionID); err != nil {
		return UploadResult{}, err
	}
	approval, err := s.campaign.GetApprovalStatus(ctx, candidate.SubmissionID)
	if err != nil {
		approval = domain.ApprovalStatusPending
	}
	existing, err := s.assets.GetBySubmissionAndChecksum(ctx, candidate.SubmissionID, checksum)
	if err == nil {
		out := UploadResult{AssetID: existing.AssetID, UploadURL: signedUploadURL(existing.AssetID), ExpiresIn: 3600}
		payload, marshalErr := json.Marshal(out)
		if marshalErr != nil {
			return UploadResult{}, marshalErr
		}
		if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 201, payload, s.nowFn()); err != nil {
			return UploadResult{}, err
		}
		completed = true
		return out, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return UploadResult{}, err
	}

	assetID := uuid.NewString()
	asset := domain.MediaAsset{
		AssetID:          assetID,
		SubmissionID:     candidate.SubmissionID,
		OriginalFilename: candidate.FileName,
		MIMEType:         candidate.MIMEType,
		FileSize:         candidate.FileSize,
		SourceS3URL:      fmt.Sprintf("s3://media-raw/%s", assetID),
		Status:           domain.AssetStatusProcessing,
		ApprovalStatus:   domain.NormalizeApprovalStatus(string(approval)),
		ChecksumSHA256:   checksum,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.assets.Create(ctx, asset); err != nil {
		return UploadResult{}, err
	}
	jobs := domain.NewDefaultJobs(asset.AssetID, now)
	if err := s.jobs.CreateMany(ctx, jobs); err != nil {
		return UploadResult{}, err
	}
	for _, job := range jobs {
		if err := s.queue.Enqueue(ctx, job.JobID); err != nil {
			asset.Status = domain.AssetStatusFailed
			asset.LastErrorCode = "queue_unavailable"
			asset.LastErrorMessage = err.Error()
			asset.UpdatedAt = s.nowFn()
			_ = s.assets.Update(ctx, asset)
			_ = s.campaign.UpdateMediaStatus(ctx, asset.SubmissionID, asset.AssetID, string(asset.Status), asset.LastErrorCode)
			return UploadResult{}, err
		}
	}
	out := UploadResult{AssetID: asset.AssetID, UploadURL: signedUploadURL(asset.AssetID), ExpiresIn: 3600}
	payload, err := json.Marshal(out)
	if err != nil {
		return UploadResult{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 201, payload, s.nowFn()); err != nil {
		return UploadResult{}, err
	}
	completed = true
	return out, nil
}

func (s *Service) GetAssetStatus(ctx context.Context, actor Actor, assetID string) (contracts.AssetStatusResponse, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return contracts.AssetStatusResponse{}, domain.ErrUnauthorized
	}
	asset, err := s.assets.GetByID(ctx, strings.TrimSpace(assetID))
	if err != nil {
		return contracts.AssetStatusResponse{}, err
	}
	outputs, err := s.outputs.ListByAsset(ctx, asset.AssetID)
	if err != nil {
		return contracts.AssetStatusResponse{}, err
	}
	thumbs, err := s.thumbnails.ListByAsset(ctx, asset.AssetID)
	if err != nil {
		return contracts.AssetStatusResponse{}, err
	}
	outRows := make([]contracts.OutputDTO, 0, len(outputs))
	for _, item := range outputs {
		outRows = append(outRows, contracts.OutputDTO{Profile: string(item.Profile), AspectRatio: string(item.AspectRatio), URL: item.S3URL})
	}
	thumbRows := make([]contracts.ThumbnailDTO, 0, len(thumbs))
	for _, item := range thumbs {
		thumbRows = append(thumbRows, contracts.ThumbnailDTO{Position: item.Position, AspectRatio: string(item.AspectRatio), URL: item.S3URL})
	}
	return contracts.AssetStatusResponse{
		AssetID:    asset.AssetID,
		Status:     string(asset.Status),
		Outputs:    outRows,
		Thumbnails: thumbRows,
		ErrorCode:  asset.LastErrorCode,
		Error:      asset.LastErrorMessage,
	}, nil
}

func (s *Service) RetryAsset(ctx context.Context, actor Actor, input RetryAssetInput) (contracts.RetryResponse, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return contracts.RetryResponse{}, domain.ErrUnauthorized
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if role != "admin" {
		return contracts.RetryResponse{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return contracts.RetryResponse{}, domain.ErrIdempotencyRequired
	}
	assetID := strings.TrimSpace(input.AssetID)
	if err := validateRetryIdempotencyKey(actor.IdempotencyKey, assetID); err != nil {
		return contracts.RetryResponse{}, err
	}
	now := s.nowFn()
	requestHash := hashPayload(RetryAssetInput{AssetID: assetID})
	rec, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return contracts.RetryResponse{}, err
	}
	if rec != nil {
		if rec.RequestHash != requestHash {
			return contracts.RetryResponse{}, domain.ErrIdempotencyConflict
		}
		if len(rec.ResponseBody) == 0 {
			return contracts.RetryResponse{}, domain.ErrIdempotencyInFlight
		}
		var out contracts.RetryResponse
		if err := json.Unmarshal(rec.ResponseBody, &out); err != nil {
			return contracts.RetryResponse{}, err
		}
		return out, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return contracts.RetryResponse{}, err
	}
	completed := false
	defer func() {
		if !completed {
			_ = s.idempotency.Release(ctx, actor.IdempotencyKey)
		}
	}()

	asset, err := s.assets.GetByID(ctx, assetID)
	if err != nil {
		return contracts.RetryResponse{}, err
	}
	if asset.Status == domain.AssetStatusCompleted {
		response := contracts.RetryResponse{AssetID: asset.AssetID, JobsRestarted: 0}
		payload, err := json.Marshal(response)
		if err != nil {
			return contracts.RetryResponse{}, err
		}
		if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 200, payload, s.nowFn()); err != nil {
			return contracts.RetryResponse{}, err
		}
		completed = true
		return response, nil
	}
	failed, err := s.jobs.ListFailedByAsset(ctx, asset.AssetID)
	if err != nil {
		return contracts.RetryResponse{}, err
	}
	if len(failed) == 0 {
		return contracts.RetryResponse{}, domain.ErrConflict
	}
	count := 0
	for _, job := range failed {
		job.Status = domain.JobStatusQueued
		job.ErrorMessage = ""
		job.QueuedAt = s.nowFn()
		if err := s.jobs.Update(ctx, job); err != nil {
			return contracts.RetryResponse{}, err
		}
		if err := s.queue.Enqueue(ctx, job.JobID); err != nil {
			return contracts.RetryResponse{}, err
		}
		count++
	}
	asset.Status = domain.AssetStatusProcessing
	asset.LastErrorCode = ""
	asset.LastErrorMessage = ""
	asset.UpdatedAt = s.nowFn()
	if err := s.assets.Update(ctx, asset); err != nil {
		return contracts.RetryResponse{}, err
	}
	response := contracts.RetryResponse{AssetID: asset.AssetID, JobsRestarted: count}
	payload, err := json.Marshal(response)
	if err != nil {
		return contracts.RetryResponse{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 200, payload, s.nowFn()); err != nil {
		return contracts.RetryResponse{}, err
	}
	completed = true
	return response, nil
}

func (s *Service) GetPreviewURL(ctx context.Context, assetID string, expirySeconds int32) (PreviewResult, error) {
	asset, err := s.assets.GetByID(ctx, strings.TrimSpace(assetID))
	if err != nil {
		return PreviewResult{}, err
	}
	outputs, err := s.outputs.ListByAsset(ctx, asset.AssetID)
	if err != nil {
		return PreviewResult{}, err
	}
	if len(outputs) == 0 {
		return PreviewResult{}, domain.ErrNotFound
	}
	if expirySeconds <= 0 {
		expirySeconds = 3600
	}
	return PreviewResult{PreviewURL: outputs[0].S3URL, ExpiresAt: s.nowFn().Add(time.Duration(expirySeconds) * time.Second).Unix()}, nil
}

func (s *Service) GetAssetMetadata(ctx context.Context, assetID string) (MetadataResult, error) {
	asset, err := s.assets.GetByID(ctx, strings.TrimSpace(assetID))
	if err != nil {
		return MetadataResult{}, err
	}
	return MetadataResult{
		AssetID:         asset.AssetID,
		ContentType:     asset.MIMEType,
		FileSizeBytes:   asset.FileSize,
		Width:           1920,
		Height:          1080,
		DurationSeconds: float64(asset.DurationSeconds),
		Codec:           "h264",
	}, nil
}

func signedUploadURL(assetID string) string {
	return fmt.Sprintf("https://s3.aws.com/upload/%s", assetID)
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

func validateUploadIdempotencyKey(key, submissionID, checksum string) error {
	parts := strings.Split(strings.TrimSpace(key), ":")
	if len(parts) != 3 || parts[0] != "media-upload" {
		return domain.ErrInvalidInput
	}
	if parts[1] != submissionID {
		return domain.ErrInvalidInput
	}
	if parts[2] == "" || parts[2] != checksum {
		return domain.ErrInvalidInput
	}
	return nil
}

func validateRetryIdempotencyKey(key, assetID string) error {
	parts := strings.Split(strings.TrimSpace(key), ":")
	if len(parts) != 3 || parts[0] != "media-retry" {
		return domain.ErrInvalidInput
	}
	if parts[1] != assetID {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(parts[2]) == "" {
		return domain.ErrInvalidInput
	}
	return nil
}
