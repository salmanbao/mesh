package postgres

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M78-logging-service/internal/ports"
)

type Repositories struct {
	Logs        *LogEventRepository
	Alerts      *AlertRuleRepository
	Exports     *ExportRepository
	Audits      *AuditRepository
	Metrics     *MetricsRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Logs:        &LogEventRepository{rows: []domain.LogEvent{}},
		Alerts:      &AlertRuleRepository{rows: map[string]domain.AlertRule{}, order: []string{}},
		Exports:     &ExportRepository{rows: map[string]domain.LogExport{}, order: []string{}},
		Audits:      &AuditRepository{rows: []domain.AuditLog{}},
		Metrics:     &MetricsRepository{counters: map[string]ports.MetricCounterPoint{}, histograms: map[string]ports.MetricHistogramPoint{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type LogEventRepository struct {
	mu   sync.Mutex
	rows []domain.LogEvent
}

func (r *LogEventRepository) InsertBatch(_ context.Context, rows []domain.LogEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, rows...)
	return nil
}

func (r *LogEventRepository) Search(_ context.Context, q domain.LogSearchQuery) ([]domain.LogEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := make([]domain.LogEvent, 0, limit)
	qService := strings.TrimSpace(q.Service)
	qLevel := strings.TrimSpace(q.Level)
	qText := strings.ToLower(strings.TrimSpace(q.Q))
	for i := len(r.rows) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.rows[i]
		if qService != "" && row.Service != qService {
			continue
		}
		if qLevel != "" && row.Level != qLevel {
			continue
		}
		if q.From != nil && row.Timestamp.Before(*q.From) {
			continue
		}
		if q.To != nil && row.Timestamp.After(*q.To) {
			continue
		}
		if qText != "" {
			hay := strings.ToLower(row.Message + " " + row.ErrorCode + " " + row.TraceID)
			if !strings.Contains(hay, qText) {
				continue
			}
		}
		out = append(out, row)
	}
	return out, nil
}

type AlertRuleRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.AlertRule
	order []string
}

func (r *AlertRuleRepository) Create(_ context.Context, row domain.AlertRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.RuleID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.RuleID] = row
	r.order = append(r.order, row.RuleID)
	return nil
}

func (r *AlertRuleRepository) List(_ context.Context, onlyEnabled bool) ([]domain.AlertRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.AlertRule, 0, len(r.rows))
	for _, id := range r.order {
		row := r.rows[id]
		if onlyEnabled && !row.Enabled {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

type ExportRepository struct {
	mu    sync.Mutex
	rows  map[string]domain.LogExport
	order []string
}

func (r *ExportRepository) Create(_ context.Context, row domain.LogExport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.ExportID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.ExportID] = row
	r.order = append(r.order, row.ExportID)
	return nil
}

func (r *ExportRepository) GetByID(_ context.Context, exportID string) (domain.LogExport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[strings.TrimSpace(exportID)]
	if !ok {
		return domain.LogExport{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *ExportRepository) List(_ context.Context, limit int) ([]domain.LogExport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := make([]domain.LogExport, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.rows[r.order[i]])
	}
	return out, nil
}

func (r *ExportRepository) Update(_ context.Context, row domain.LogExport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.ExportID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[row.ExportID] = row
	return nil
}

type AuditRepository struct {
	mu   sync.Mutex
	rows []domain.AuditLog
}

func (r *AuditRepository) Create(_ context.Context, row domain.AuditLog) error {
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
	out := domain.AuditQueryResult{Logs: make([]domain.AuditLog, 0, limit)}
	for i := len(r.rows) - 1; i >= 0 && len(out.Logs) < limit; i-- {
		row := r.rows[i]
		if q.ActorID != "" && row.ActorID != q.ActorID {
			continue
		}
		if q.ActionType != "" && row.ActionType != q.ActionType {
			continue
		}
		out.Logs = append(out.Logs, row)
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
