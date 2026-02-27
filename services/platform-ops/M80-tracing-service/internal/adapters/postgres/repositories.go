package postgres

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M80-tracing-service/internal/ports"
)

type Repositories struct {
	Traces      *TraceRepository
	Spans       *SpanRepository
	SpanTags    *SpanTagRepository
	Policies    *SamplingPolicyRepository
	Exports     *ExportRepository
	AuditLogs   *AuditLogRepository
	Metrics     *MetricsRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Traces:      &TraceRepository{rows: map[string]domain.TraceRecord{}},
		Spans:       &SpanRepository{rows: map[string]domain.SpanRecord{}},
		SpanTags:    &SpanTagRepository{rows: map[string]domain.SpanTag{}},
		Policies:    &SamplingPolicyRepository{rows: map[string]domain.SamplingPolicy{}},
		Exports:     &ExportRepository{rows: map[string]domain.ExportJob{}, order: []string{}},
		AuditLogs:   &AuditLogRepository{rows: []domain.AuditLog{}},
		Metrics:     &MetricsRepository{counters: map[string]ports.MetricCounterPoint{}, histograms: map[string]ports.MetricHistogramPoint{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type TraceRepository struct {
	mu   sync.Mutex
	rows map[string]domain.TraceRecord
}

func (r *TraceRepository) UpsertFromSpans(_ context.Context, spans []domain.SpanRecord, environment string) ([]domain.TraceRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := map[string]struct{}{}
	for _, sp := range spans {
		if strings.TrimSpace(sp.TraceID) == "" {
			continue
		}
		row := r.rows[sp.TraceID]
		now := time.Now().UTC()
		if row.TraceID == "" {
			row = domain.TraceRecord{TraceID: sp.TraceID, RootService: sp.ServiceName, StartTime: sp.StartTime, EndTime: sp.EndTime, Error: sp.Error, Environment: environment, CreatedAt: now}
		}
		if row.StartTime.IsZero() || sp.StartTime.Before(row.StartTime) {
			row.StartTime = sp.StartTime
			if strings.TrimSpace(sp.ServiceName) != "" {
				row.RootService = sp.ServiceName
			}
		}
		if row.EndTime.IsZero() || sp.EndTime.After(row.EndTime) {
			row.EndTime = sp.EndTime
		}
		row.Error = row.Error || sp.Error || sp.HTTPStatusCode >= 500
		if row.Environment == "" && environment != "" {
			row.Environment = environment
		}
		row.DurationMS = row.EndTime.Sub(row.StartTime).Milliseconds()
		if row.DurationMS < 0 {
			row.DurationMS = 0
		}
		row.UpdatedAt = now
		r.rows[row.TraceID] = row
		seen[row.TraceID] = struct{}{}
	}
	out := make([]domain.TraceRecord, 0, len(seen))
	for id := range seen {
		out = append(out, r.rows[id])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TraceID < out[j].TraceID })
	return out, nil
}

func (r *TraceRepository) GetByID(_ context.Context, traceID string) (domain.TraceRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(traceID)]
	if !ok {
		return domain.TraceRecord{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *TraceRepository) Search(_ context.Context, q domain.TraceSearchQuery) ([]domain.TraceSearchHit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := make([]domain.TraceSearchHit, 0)
	for _, row := range r.rows {
		if q.TraceID != "" && row.TraceID != q.TraceID {
			continue
		}
		if q.ServiceName != "" && !strings.EqualFold(row.RootService, q.ServiceName) {
			continue
		}
		if q.ErrorOnly != nil && *q.ErrorOnly && !row.Error {
			continue
		}
		if q.DurationGTMS != nil && row.DurationMS <= *q.DurationGTMS {
			continue
		}
		out = append(out, domain.TraceSearchHit{TraceID: row.TraceID, DurationMS: row.DurationMS, Error: row.Error})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DurationMS == out[j].DurationMS {
			return out[i].TraceID < out[j].TraceID
		}
		return out[i].DurationMS > out[j].DurationMS
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *TraceRepository) Count(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.rows), nil
}

type SpanRepository struct {
	mu   sync.Mutex
	rows map[string]domain.SpanRecord
}

func (r *SpanRepository) UpsertBatch(_ context.Context, spans []domain.SpanRecord) (int, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inserted, duplicates := 0, 0
	for _, sp := range spans {
		if _, ok := r.rows[sp.SpanID]; ok {
			duplicates++
			continue
		}
		r.rows[sp.SpanID] = sp
		inserted++
	}
	return inserted, duplicates, nil
}

func (r *SpanRepository) ListByTraceID(_ context.Context, traceID string) ([]domain.SpanRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.SpanRecord, 0)
	for _, row := range r.rows {
		if row.TraceID == strings.TrimSpace(traceID) {
			out = append(out, row)
		}
	}
	if len(out) == 0 {
		return nil, domain.ErrNotFound
	}
	return domain.NormalizeSpansForTimeline(out), nil
}

type SpanTagRepository struct {
	mu   sync.Mutex
	rows map[string]domain.SpanTag
}

func (r *SpanTagRepository) ReplaceForSpans(_ context.Context, tags []domain.SpanTag) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range tags {
		r.rows[t.TagID] = t
	}
	return nil
}

func (r *SpanTagRepository) ListByTraceID(_ context.Context, traceID string) ([]domain.SpanTag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.SpanTag, 0)
	for _, row := range r.rows {
		if strings.HasPrefix(row.TagID, strings.TrimSpace(traceID)+":") {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TagID < out[j].TagID })
	return out, nil
}

type SamplingPolicyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.SamplingPolicy
}

func (r *SamplingPolicyRepository) Create(_ context.Context, row domain.SamplingPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.PolicyID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.PolicyID] = row
	return nil
}
func (r *SamplingPolicyRepository) List(_ context.Context) ([]domain.SamplingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.SamplingPolicy, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *SamplingPolicyRepository) GetByID(_ context.Context, policyID string) (domain.SamplingPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(policyID)]
	if !ok {
		return domain.SamplingPolicy{}, domain.ErrNotFound
	}
	return row, nil
}

type ExportRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.ExportJob
	order []string
}

func (r *ExportRepository) Create(_ context.Context, row domain.ExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.ExportID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.ExportID] = row
	r.order = append(r.order, row.ExportID)
	return nil
}
func (r *ExportRepository) GetByID(_ context.Context, exportID string) (domain.ExportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(exportID)]
	if !ok {
		return domain.ExportJob{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ExportRepository) Update(_ context.Context, row domain.ExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.ExportID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[row.ExportID] = row
	return nil
}
func (r *ExportRepository) List(_ context.Context, limit int) ([]domain.ExportJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := make([]domain.ExportJob, 0, limit)
	for i := len(r.order) - 1; i >= 0; i-- {
		out = append(out, r.rows[r.order[i]])
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

type AuditLogRepository struct {
	mu   sync.Mutex
	rows []domain.AuditLog
}

func (r *AuditLogRepository) Create(_ context.Context, row domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *AuditLogRepository) List(_ context.Context, limit int) ([]domain.AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > len(r.rows) {
		limit = len(r.rows)
	}
	out := make([]domain.AuditLog, 0, limit)
	for i := len(r.rows) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.rows[i])
	}
	return out, nil
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
	out := ports.MetricsSnapshot{Counters: make([]ports.MetricCounterPoint, 0, len(r.counters)), Histograms: make([]ports.MetricHistogramPoint, 0, len(r.histograms))}
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
