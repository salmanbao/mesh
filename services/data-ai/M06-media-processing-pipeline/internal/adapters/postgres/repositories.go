package postgres

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/ports"
)

type Repositories struct {
	Assets      *AssetRepository
	Jobs        *JobRepository
	Outputs     *OutputRepository
	Thumbnails  *ThumbnailRepository
	Watermarks  *WatermarkRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Assets:      &AssetRepository{records: map[string]domain.MediaAsset{}, byChecksum: map[string]string{}},
		Jobs:        &JobRepository{records: map[string]domain.MediaJob{}, byAsset: map[string][]string{}},
		Outputs:     &OutputRepository{records: map[string]domain.MediaOutput{}, byAsset: map[string][]string{}, byUnique: map[string]string{}},
		Thumbnails:  &ThumbnailRepository{records: map[string]domain.MediaThumbnail{}, byAsset: map[string][]string{}, byUnique: map[string]string{}},
		Watermarks:  &WatermarkRepository{records: map[string]domain.WatermarkRecord{}},
		Idempotency: &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{records: map[string]dedupRecord{}},
	}
}

type AssetRepository struct {
	mu         sync.RWMutex
	records    map[string]domain.MediaAsset
	byChecksum map[string]string
}

func submissionChecksumKey(submissionID, checksum string) string {
	return submissionID + "::" + checksum
}

func (r *AssetRepository) Create(_ context.Context, asset domain.MediaAsset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[asset.AssetID] = asset
	r.byChecksum[submissionChecksumKey(asset.SubmissionID, asset.ChecksumSHA256)] = asset.AssetID
	return nil
}

func (r *AssetRepository) GetByID(_ context.Context, assetID string) (domain.MediaAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	asset, ok := r.records[assetID]
	if !ok {
		return domain.MediaAsset{}, domain.ErrNotFound
	}
	return asset, nil
}

func (r *AssetRepository) GetBySubmissionAndChecksum(_ context.Context, submissionID, checksum string) (domain.MediaAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	assetID, ok := r.byChecksum[submissionChecksumKey(submissionID, checksum)]
	if !ok {
		return domain.MediaAsset{}, domain.ErrNotFound
	}
	asset, exists := r.records[assetID]
	if !exists {
		return domain.MediaAsset{}, domain.ErrNotFound
	}
	return asset, nil
}

func (r *AssetRepository) Update(_ context.Context, asset domain.MediaAsset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[asset.AssetID] = asset
	r.byChecksum[submissionChecksumKey(asset.SubmissionID, asset.ChecksumSHA256)] = asset.AssetID
	return nil
}

type JobRepository struct {
	mu      sync.RWMutex
	records map[string]domain.MediaJob
	byAsset map[string][]string
}

func (r *JobRepository) CreateMany(_ context.Context, jobs []domain.MediaJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, job := range jobs {
		r.records[job.JobID] = job
		r.byAsset[job.AssetID] = append(r.byAsset[job.AssetID], job.JobID)
	}
	return nil
}

func (r *JobRepository) GetByID(_ context.Context, jobID string) (domain.MediaJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.records[jobID]
	if !ok {
		return domain.MediaJob{}, domain.ErrNotFound
	}
	return job, nil
}

func (r *JobRepository) Update(_ context.Context, job domain.MediaJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[job.JobID] = job
	return nil
}

func (r *JobRepository) ListByAsset(_ context.Context, assetID string) ([]domain.MediaJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byAsset[assetID]
	out := make([]domain.MediaJob, 0, len(ids))
	for _, id := range ids {
		if job, ok := r.records[id]; ok {
			out = append(out, job)
		}
	}
	slices.SortFunc(out, func(a, b domain.MediaJob) int {
		return a.QueuedAt.Compare(b.QueuedAt)
	})
	return out, nil
}

func (r *JobRepository) ListFailedByAsset(ctx context.Context, assetID string) ([]domain.MediaJob, error) {
	jobs, _ := r.ListByAsset(ctx, assetID)
	out := make([]domain.MediaJob, 0, len(jobs))
	for _, job := range jobs {
		if job.Status == domain.JobStatusFailed {
			out = append(out, job)
		}
	}
	return out, nil
}

type OutputRepository struct {
	mu       sync.RWMutex
	records  map[string]domain.MediaOutput
	byAsset  map[string][]string
	byUnique map[string]string
}

func outputKey(assetID string, profile domain.OutputProfile, aspectRatio domain.AspectRatio) string {
	return assetID + "::" + string(profile) + "::" + string(aspectRatio)
}

func (r *OutputRepository) Upsert(_ context.Context, output domain.MediaOutput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	unique := outputKey(output.AssetID, output.Profile, output.AspectRatio)
	if existingID, ok := r.byUnique[unique]; ok {
		output.OutputID = existingID
		r.records[existingID] = output
		return nil
	}
	r.records[output.OutputID] = output
	r.byAsset[output.AssetID] = append(r.byAsset[output.AssetID], output.OutputID)
	r.byUnique[unique] = output.OutputID
	return nil
}

func (r *OutputRepository) ListByAsset(_ context.Context, assetID string) ([]domain.MediaOutput, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byAsset[assetID]
	out := make([]domain.MediaOutput, 0, len(ids))
	for _, id := range ids {
		if item, ok := r.records[id]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

type ThumbnailRepository struct {
	mu       sync.RWMutex
	records  map[string]domain.MediaThumbnail
	byAsset  map[string][]string
	byUnique map[string]string
}

func thumbnailKey(assetID string, aspectRatio domain.AspectRatio, position string) string {
	return assetID + "::" + string(aspectRatio) + "::" + position
}

func (r *ThumbnailRepository) Upsert(_ context.Context, thumbnail domain.MediaThumbnail) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	unique := thumbnailKey(thumbnail.AssetID, thumbnail.AspectRatio, thumbnail.Position)
	if existingID, ok := r.byUnique[unique]; ok {
		thumbnail.ThumbnailID = existingID
		r.records[existingID] = thumbnail
		return nil
	}
	r.records[thumbnail.ThumbnailID] = thumbnail
	r.byAsset[thumbnail.AssetID] = append(r.byAsset[thumbnail.AssetID], thumbnail.ThumbnailID)
	r.byUnique[unique] = thumbnail.ThumbnailID
	return nil
}

func (r *ThumbnailRepository) ListByAsset(_ context.Context, assetID string) ([]domain.MediaThumbnail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byAsset[assetID]
	out := make([]domain.MediaThumbnail, 0, len(ids))
	for _, id := range ids {
		if item, ok := r.records[id]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

type WatermarkRepository struct {
	mu      sync.RWMutex
	records map[string]domain.WatermarkRecord
}

func (r *WatermarkRepository) Upsert(_ context.Context, record domain.WatermarkRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[record.AssetID] = record
	return nil
}

func (r *WatermarkRepository) GetByAsset(_ context.Context, assetID string) (domain.WatermarkRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[assetID]
	if !ok {
		return domain.WatermarkRecord{}, domain.ErrNotFound
	}
	return record, nil
}

type IdempotencyRepository struct {
	mu      sync.Mutex
	records map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.records, key)
		return nil, nil
	}
	clone := rec
	return &clone, nil
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.records[key]; ok && time.Now().UTC().Before(existing.ExpiresAt) {
		if existing.RequestHash != requestHash {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	r.records[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[key]
	if !ok {
		return domain.ErrNotFound
	}
	rec.ResponseCode = responseCode
	rec.ResponseBody = slices.Clone(responseBody)
	if at.After(rec.ExpiresAt) {
		rec.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.records[key] = rec
	return nil
}

func (r *IdempotencyRepository) Release(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, key)
	return nil
}

type dedupRecord struct {
	EventType string
	ExpiresAt time.Time
}

type EventDedupRepository struct {
	mu      sync.Mutex
	records map[string]dedupRecord
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.records[eventID]
	if !ok {
		return false, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.records, eventID)
		return false, nil
	}
	return true, nil
}

func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[eventID] = dedupRecord{EventType: eventType, ExpiresAt: expiresAt}
	return nil
}
