package postgres

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/ports"
)

type Repositories struct {
	Topics      *TopicRepository
	ACLs        *ACLRepository
	Offsets     *OffsetRepository
	Schemas     *SchemaRepository
	DLQ         *DLQRepository
	Metrics     *MetricsRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Topics:      &TopicRepository{rows: map[string]domain.Topic{}, byName: map[string]string{}},
		ACLs:        &ACLRepository{rows: map[string]domain.ACLRecord{}, order: []string{}},
		Offsets:     &OffsetRepository{rows: map[string]domain.ConsumerOffsetAudit{}, order: []string{}},
		Schemas:     &SchemaRepository{rows: map[string][]domain.SchemaRecord{}},
		DLQ:         &DLQRepository{rows: map[string]domain.DLQMessage{}, order: []string{}},
		Metrics:     &MetricsRepository{counters: map[string]ports.MetricCounterPoint{}, histograms: map[string]ports.MetricHistogramPoint{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type TopicRepository struct {
	mu     sync.Mutex
	rows   map[string]domain.Topic
	byName map[string]string
}

func (r *TopicRepository) Create(_ context.Context, row domain.Topic) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byName[row.TopicName]; ok {
		return domain.ErrConflict
	}
	r.rows[row.ID] = row
	r.byName[row.TopicName] = row.ID
	return nil
}

func (r *TopicRepository) GetByName(_ context.Context, topicName string) (domain.Topic, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byName[strings.TrimSpace(topicName)]
	if !ok {
		return domain.Topic{}, domain.ErrNotFound
	}
	return r.rows[id], nil
}

func (r *TopicRepository) List(_ context.Context, limit int) ([]domain.Topic, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	out := make([]domain.Topic, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TopicName < out[j].TopicName })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type ACLRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.ACLRecord
	order []string
}

func (r *ACLRepository) Create(_ context.Context, row domain.ACLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range r.order {
		existing := r.rows[id]
		if existing.Principal == row.Principal &&
			existing.ResourceType == row.ResourceType &&
			existing.ResourceName == row.ResourceName &&
			existing.PatternType == row.PatternType &&
			strings.Join(existing.Operations, ",") == strings.Join(row.Operations, ",") {
			return domain.ErrConflict
		}
	}
	r.rows[row.ID] = row
	r.order = append(r.order, row.ID)
	return nil
}

func (r *ACLRepository) List(_ context.Context, limit int) ([]domain.ACLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := make([]domain.ACLRecord, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.rows[r.order[i]])
	}
	return out, nil
}

type OffsetRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.ConsumerOffsetAudit
	order []string
}

func (r *OffsetRepository) Create(_ context.Context, row domain.ConsumerOffsetAudit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[row.ID] = row
	r.order = append(r.order, row.ID)
	return nil
}

func (r *OffsetRepository) ListByGroup(_ context.Context, groupID string, limit int) ([]domain.ConsumerOffsetAudit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	groupID = strings.TrimSpace(groupID)
	out := make([]domain.ConsumerOffsetAudit, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.rows[r.order[i]]
		if groupID != "" && row.GroupID != groupID {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

type SchemaRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.SchemaRecord
}

func (r *SchemaRepository) Register(_ context.Context, row domain.SchemaRecord) (domain.SchemaRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.rows[row.Subject]
	row.Version = len(list) + 1
	r.rows[row.Subject] = append(list, row)
	return row, nil
}

func (r *SchemaRepository) GetLatestBySubject(_ context.Context, subject string) (domain.SchemaRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.rows[strings.TrimSpace(subject)]
	if len(list) == 0 {
		return domain.SchemaRecord{}, domain.ErrNotFound
	}
	return list[len(list)-1], nil
}

func (r *SchemaRepository) List(_ context.Context, limit int) ([]domain.SchemaRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := make([]domain.SchemaRecord, 0)
	for _, list := range r.rows {
		out = append(out, list...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type DLQRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.DLQMessage
	order []string
}

func (r *DLQRepository) Create(_ context.Context, row domain.DLQMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.ID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.ID] = row
	r.order = append(r.order, row.ID)
	return nil
}

func (r *DLQRepository) Query(_ context.Context, q domain.DLQQuery) ([]domain.DLQMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := q.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	out := make([]domain.DLQMessage, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.rows[r.order[i]]
		if q.SourceTopic != "" && row.SourceTopic != q.SourceTopic {
			continue
		}
		if q.ConsumerGroup != "" && row.ConsumerGroup != q.ConsumerGroup {
			continue
		}
		if q.ErrorType != "" && row.ErrorType != q.ErrorType {
			continue
		}
		if !q.IncludeReplayed && row.ReplayedAt != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *DLQRepository) MarkReplayed(_ context.Context, ids []string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		row, ok := r.rows[id]
		if !ok {
			continue
		}
		t := at
		row.ReplayedAt = &t
		r.rows[id] = row
	}
	return nil
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
