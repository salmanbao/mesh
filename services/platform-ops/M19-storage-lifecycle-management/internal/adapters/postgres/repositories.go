package postgres

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/ports"
)

type Repositories struct {
	Policies    *PolicyRepository
	Lifecycle   *LifecycleRepository
	Batches     *DeletionBatchRepository
	Audits      *AuditRepository
	Metrics     *MetricsRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Policies:    &PolicyRepository{rows: map[string]domain.StoragePolicy{}},
		Lifecycle:   &LifecycleRepository{rows: map[string]domain.LifecycleFile{}},
		Batches:     &DeletionBatchRepository{rows: map[string]domain.DeletionBatch{}, order: []string{}},
		Audits:      &AuditRepository{rows: []domain.AuditRecord{}},
		Metrics:     &MetricsRepository{counters: map[string]ports.MetricCounterPoint{}, histograms: map[string]ports.MetricHistogramPoint{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type PolicyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.StoragePolicy
}

func (r *PolicyRepository) Create(_ context.Context, row domain.StoragePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.PolicyID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.PolicyID] = row
	return nil
}

func (r *PolicyRepository) List(_ context.Context) ([]domain.StoragePolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.StoragePolicy, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

type LifecycleRepository struct {
	mu   sync.Mutex
	rows map[string]domain.LifecycleFile
}

func (r *LifecycleRepository) Upsert(_ context.Context, row domain.LifecycleFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.rows[row.FileID]
	if ok {
		if existing.CreatedAt.IsZero() {
			existing.CreatedAt = row.CreatedAt
		}
		if row.CampaignID != "" {
			existing.CampaignID = row.CampaignID
		}
		if row.SubmissionID != "" {
			existing.SubmissionID = row.SubmissionID
		}
		if row.FileSizeBytes > 0 {
			existing.FileSizeBytes = row.FileSizeBytes
		}
		if row.SourceBucket != "" {
			existing.SourceBucket = row.SourceBucket
		}
		if row.SourceKey != "" {
			existing.SourceKey = row.SourceKey
		}
		if row.DestinationBucket != "" {
			existing.DestinationBucket = row.DestinationBucket
		}
		if row.DestinationKey != "" {
			existing.DestinationKey = row.DestinationKey
		}
		if row.ChecksumMD5 != "" {
			existing.ChecksumMD5 = row.ChecksumMD5
		}
		if row.StorageTier != "" {
			existing.StorageTier = row.StorageTier
		}
		if row.Status != "" {
			existing.Status = row.Status
		}
		existing.LegalHold = existing.LegalHold || row.LegalHold
		if !row.UpdatedAt.IsZero() {
			existing.UpdatedAt = row.UpdatedAt
		} else {
			existing.UpdatedAt = time.Now().UTC()
		}
		r.rows[row.FileID] = existing
		return nil
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	}
	if row.StorageTier == "" {
		row.StorageTier = domain.TierStandard
	}
	if row.Status == "" {
		row.Status = domain.FileStatusUploaded
	}
	r.rows[row.FileID] = row
	return nil
}

func (r *LifecycleRepository) GetByID(_ context.Context, fileID string) (domain.LifecycleFile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(fileID)]
	if !ok {
		return domain.LifecycleFile{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *LifecycleRepository) ListByCampaign(_ context.Context, campaignID string) ([]domain.LifecycleFile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.LifecycleFile, 0)
	for _, row := range r.rows {
		if row.CampaignID == strings.TrimSpace(campaignID) {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FileID < out[j].FileID })
	return out, nil
}

func (r *LifecycleRepository) AnalyticsSummary(_ context.Context) (domain.AnalyticsSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	byTier := map[string]int64{}
	var totalObjects int64
	var monthlyCost float64
	for _, row := range r.rows {
		tier := row.StorageTier
		if tier == "" {
			tier = domain.TierStandard
		}
		byTier[tier]++
		totalObjects++
		gb := float64(row.FileSizeBytes) / (1024 * 1024 * 1024)
		switch tier {
		case domain.TierGlacierDeepArchive, domain.TierGlacier:
			monthlyCost += gb * 0.004
		default:
			monthlyCost += gb * 0.023
		}
	}
	return domain.AnalyticsSummary{
		TotalObjects: totalObjects,
		ByTier:       byTier,
		MonthlyCost:  monthlyCost,
		LastRunAt:    time.Now().UTC(),
	}, nil
}

type DeletionBatchRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.DeletionBatch
	order []string
}

func (r *DeletionBatchRepository) Create(_ context.Context, row domain.DeletionBatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.BatchID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.BatchID] = row
	r.order = append(r.order, row.BatchID)
	return nil
}

func (r *DeletionBatchRepository) GetByID(_ context.Context, batchID string) (domain.DeletionBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(batchID)]
	if !ok {
		return domain.DeletionBatch{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *DeletionBatchRepository) List(_ context.Context, limit int) ([]domain.DeletionBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := make([]domain.DeletionBatch, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.rows[r.order[i]])
	}
	return out, nil
}

type AuditRepository struct {
	mu   sync.Mutex
	rows []domain.AuditRecord
}

func (r *AuditRepository) Create(_ context.Context, row domain.AuditRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}

func (r *AuditRepository) Query(_ context.Context, q domain.AuditQuery) (domain.AuditQueryResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	result := domain.AuditQueryResult{Records: make([]domain.AuditRecord, 0, limit)}
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		if q.FileID != "" && row.FileID != q.FileID {
			continue
		}
		if q.CampaignID != "" && row.CampaignID != q.CampaignID {
			continue
		}
		if q.Action != "" && row.Action != q.Action {
			continue
		}
		if q.StartDate != nil && row.InitiatedAt.Before(*q.StartDate) {
			continue
		}
		if q.EndDate != nil && row.InitiatedAt.After(*q.EndDate) {
			continue
		}
		result.TotalSizeFreed += row.FileSizeBytes
		if row.Action == "soft_delete" || row.Action == "hard_delete" {
			result.TotalFilesDeleted++
		}
		if len(result.Records) < limit {
			result.Records = append(result.Records, row)
		}
	}
	return result, nil
}

type MetricsRepository struct {
	mu         sync.Mutex
	counters   map[string]ports.MetricCounterPoint
	histograms map[string]ports.MetricHistogramPoint
}

func (r *MetricsRepository) IncCounter(_ context.Context, name string, labels map[string]string, delta float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := metricKey(name, labels)
	pt := r.counters[k]
	if pt.Name == "" {
		pt = ports.MetricCounterPoint{Name: name, Labels: copyLabels(labels)}
	}
	pt.Value += delta
	r.counters[k] = pt
	return nil
}

func (r *MetricsRepository) ObserveHistogram(_ context.Context, name string, labels map[string]string, value float64, buckets []float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := metricKey(name, labels)
	pt := r.histograms[k]
	if pt.Name == "" {
		pt = ports.MetricHistogramPoint{Name: name, Labels: copyLabels(labels), Buckets: map[string]float64{}}
	}
	for _, le := range buckets {
		if value <= le {
			pt.Buckets[strconv.FormatFloat(le, 'f', -1, 64)]++
		}
	}
	pt.Buckets["+Inf"]++
	pt.Sum += value
	pt.Count++
	r.histograms[k] = pt
	return nil
}

func (r *MetricsRepository) Snapshot(_ context.Context) (ports.MetricsSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := ports.MetricsSnapshot{
		Counters:   make([]ports.MetricCounterPoint, 0, len(r.counters)),
		Histograms: make([]ports.MetricHistogramPoint, 0, len(r.histograms)),
	}
	for _, c := range r.counters {
		out.Counters = append(out.Counters, ports.MetricCounterPoint{Name: c.Name, Labels: copyLabels(c.Labels), Value: c.Value})
	}
	for _, h := range r.histograms {
		b := map[string]float64{}
		for k, v := range h.Buckets {
			b[k] = v
		}
		out.Histograms = append(out.Histograms, ports.MetricHistogramPoint{Name: h.Name, Labels: copyLabels(h.Labels), Buckets: b, Sum: h.Sum, Count: h.Count})
	}
	return out, nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok {
		return nil, nil
	}
	if !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	cp := row
	cp.ResponseBody = append([]byte(nil), row.ResponseBody...)
	return &cp, nil
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[key]; ok {
		if row.RequestHash != requestHash {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	r.rows[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[key]
	row.ResponseCode = responseCode
	row.ResponseBody = append([]byte(nil), responseBody...)
	if row.ExpiresAt.IsZero() {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.rows[key] = row
	return nil
}

type EventDedupRepository struct {
	mu   sync.Mutex
	rows map[string]time.Time
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	exp, ok := r.rows[eventID]
	if !ok {
		return false, nil
	}
	if now.After(exp) {
		delete(r.rows, eventID)
		return false, nil
	}
	return true, nil
}

func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, _ string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[eventID] = expiresAt
	return nil
}

type OutboxRepository struct {
	mu    sync.Mutex
	rows  map[string]ports.OutboxRecord
	order []string
}

func (r *OutboxRepository) Enqueue(_ context.Context, row ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.RecordID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.RecordID] = row
	r.order = append(r.order, row.RecordID)
	return nil
}

func (r *OutboxRepository) ListPending(_ context.Context, limit int) ([]ports.OutboxRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]ports.OutboxRecord, 0, limit)
	for _, id := range r.order {
		row := r.rows[id]
		if row.SentAt != nil {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	row.SentAt = &at
	r.rows[recordID] = row
	return nil
}

func metricKey(name string, labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(name)
	for _, k := range keys {
		b.WriteString("|")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(labels[k])
	}
	return b.String()
}

func copyLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
