package postgres

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M77-config-service/internal/ports"
)

type Repositories struct {
	Keys        *ConfigKeyRepository
	Values      *ConfigValueRepository
	Versions    *ConfigVersionRepository
	Rules       *RolloutRuleRepository
	Audits      *AuditLogRepository
	Metrics     *MetricsRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Keys:        &ConfigKeyRepository{rowsByID: map[string]domain.ConfigKey{}, idByName: map[string]string{}},
		Values:      &ConfigValueRepository{rows: map[string]domain.ConfigValue{}},
		Versions:    &ConfigVersionRepository{rows: []domain.ConfigVersion{}},
		Rules:       &RolloutRuleRepository{rowsByKeyID: map[string]domain.RolloutRule{}},
		Audits:      &AuditLogRepository{rows: []domain.AuditLog{}},
		Metrics:     &MetricsRepository{counters: map[string]ports.MetricCounterPoint{}, histograms: map[string]ports.MetricHistogramPoint{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type ConfigKeyRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.ConfigKey
	idByName map[string]string
}

func (r *ConfigKeyRepository) GetByName(_ context.Context, keyName string) (domain.ConfigKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	keyName = strings.TrimSpace(keyName)
	id, ok := r.idByName[keyName]
	if !ok {
		return domain.ConfigKey{}, domain.ErrNotFound
	}
	return r.rowsByID[id], nil
}

func (r *ConfigKeyRepository) Upsert(_ context.Context, row domain.ConfigKey) (domain.ConfigKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	keyName := strings.TrimSpace(row.KeyName)
	if keyName == "" {
		return domain.ConfigKey{}, domain.ErrInvalidInput
	}
	if id, ok := r.idByName[keyName]; ok {
		existing := r.rowsByID[id]
		if existing.ValueType != row.ValueType {
			return domain.ConfigKey{}, domain.ErrConflict
		}
		return existing, nil
	}
	r.idByName[keyName] = row.KeyID
	r.rowsByID[row.KeyID] = row
	return row, nil
}

func (r *ConfigKeyRepository) Update(_ context.Context, row domain.ConfigKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.KeyID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.KeyID] = row
	r.idByName[row.KeyName] = row.KeyID
	return nil
}

func (r *ConfigKeyRepository) List(_ context.Context) ([]domain.ConfigKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.ConfigKey, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].KeyName < out[j].KeyName })
	return out, nil
}

type ConfigValueRepository struct {
	mu   sync.Mutex
	rows map[string]domain.ConfigValue
}

func (r *ConfigValueRepository) Upsert(_ context.Context, row domain.ConfigValue) (domain.ConfigValue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := valueKey(row.KeyID, row.Environment, row.ServiceScope)
	if existing, ok := r.rows[k]; ok {
		if row.ValueID == "" {
			row.ValueID = existing.ValueID
		}
	}
	r.rows[k] = row
	return row, nil
}

func (r *ConfigValueRepository) Get(_ context.Context, keyID, environment, serviceScope string) (domain.ConfigValue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[valueKey(keyID, environment, serviceScope)]
	if !ok {
		return domain.ConfigValue{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *ConfigValueRepository) ListByEnvironment(_ context.Context, environment string) ([]domain.ConfigValue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.ConfigValue, 0)
	environment = strings.TrimSpace(environment)
	for _, row := range r.rows {
		if row.Environment == environment {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].KeyID == out[j].KeyID {
			return out[i].ServiceScope < out[j].ServiceScope
		}
		return out[i].KeyID < out[j].KeyID
	})
	return out, nil
}

type ConfigVersionRepository struct {
	mu   sync.Mutex
	rows []domain.ConfigVersion
}

func (r *ConfigVersionRepository) Create(_ context.Context, row domain.ConfigVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}

func (r *ConfigVersionRepository) ListByScope(_ context.Context, keyID, environment, serviceScope string, limit int) ([]domain.ConfigVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]domain.ConfigVersion, 0, limit)
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		if row.KeyID != keyID || row.Environment != environment || row.ServiceScope != serviceScope {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *ConfigVersionRepository) GetByVersionNumber(_ context.Context, keyID, environment, serviceScope string, versionNumber int) (domain.ConfigVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.KeyID == keyID && row.Environment == environment && row.ServiceScope == serviceScope && row.VersionNumber == versionNumber {
			return row, nil
		}
	}
	return domain.ConfigVersion{}, domain.ErrNotFound
}

func (r *ConfigVersionRepository) NextVersionNumber(_ context.Context, keyID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maxVersion := 0
	for _, row := range r.rows {
		if row.KeyID == keyID && row.VersionNumber > maxVersion {
			maxVersion = row.VersionNumber
		}
	}
	return maxVersion + 1, nil
}

type RolloutRuleRepository struct {
	mu          sync.Mutex
	rowsByKeyID map[string]domain.RolloutRule
}

func (r *RolloutRuleRepository) UpsertForKey(_ context.Context, row domain.RolloutRule) (domain.RolloutRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.rowsByKeyID[row.KeyID]; ok {
		row.RuleID = existing.RuleID
		row.CreatedAt = existing.CreatedAt
	}
	r.rowsByKeyID[row.KeyID] = row
	return row, nil
}

func (r *RolloutRuleRepository) GetByKeyID(_ context.Context, keyID string) (domain.RolloutRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByKeyID[strings.TrimSpace(keyID)]
	if !ok {
		return domain.RolloutRule{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *RolloutRuleRepository) List(_ context.Context) ([]domain.RolloutRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.RolloutRule, 0, len(r.rowsByKeyID))
	for _, row := range r.rowsByKeyID {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].KeyName < out[j].KeyName })
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

func (r *AuditLogRepository) Query(_ context.Context, q domain.AuditQuery) (domain.AuditQueryResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := domain.AuditQueryResult{Logs: make([]domain.AuditLog, 0, limit)}
	for i := len(r.rows) - 1; i >= 0 && len(out.Logs) < limit; i-- {
		row := r.rows[i]
		if q.KeyName != "" && row.KeyName != q.KeyName {
			continue
		}
		if q.Environment != "" && row.Environment != q.Environment {
			continue
		}
		if q.ServiceScope != "" && row.ServiceScope != q.ServiceScope {
			continue
		}
		if q.ActorID != "" && row.ActorID != q.ActorID {
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

func valueKey(keyID, environment, serviceScope string) string {
	return strings.TrimSpace(keyID) + "|" + strings.TrimSpace(environment) + "|" + domain.NormalizeServiceScope(serviceScope)
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
