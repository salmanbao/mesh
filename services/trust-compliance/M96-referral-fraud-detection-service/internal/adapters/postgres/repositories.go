package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/ports"
)

type Repositories struct {
	ReferralEvents *ReferralEventRepository
	Decisions      *FraudDecisionRepository
	Policies       *RiskPolicyRepository
	Fingerprints   *DeviceFingerprintRepository
	Clusters       *ClusterRepository
	Disputes       *DisputeCaseRepository
	AuditLogs      *AuditLogRepository
	Idempotency    *IdempotencyRepository
	EventDedup     *EventDedupRepository
	Outbox         *OutboxRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	pol := &RiskPolicyRepository{records: map[string]domain.RiskPolicy{}}
	_ = pol.Upsert(context.Background(), domain.RiskPolicy{PolicyID: uuid.NewString(), Name: "default", Threshold: 0.8, ActionMap: map[string]string{"low": "allow", "medium": "flag", "high": "block", "critical": "block"}, IsActive: true, Version: "policy-2026-02-01", CreatedAt: now})
	return &Repositories{
		ReferralEvents: &ReferralEventRepository{byEventID: map[string]domain.ReferralEvent{}},
		Decisions:      &FraudDecisionRepository{byDecisionID: map[string]domain.FraudDecision{}, byEventID: map[string]string{}},
		Policies:       pol,
		Fingerprints:   &DeviceFingerprintRepository{byHash: map[string]fingerprintState{}},
		Clusters:       &ClusterRepository{byKey: map[string]clusterState{}},
		Disputes:       &DisputeCaseRepository{byDisputeID: map[string]domain.DisputeCase{}, byDecisionID: map[string]string{}},
		AuditLogs:      &AuditLogRepository{records: []domain.AuditLog{}},
		Idempotency:    &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:     &EventDedupRepository{records: map[string]dedupRecord{}},
		Outbox:         &OutboxRepository{records: map[string]ports.OutboxRecord{}},
	}
}

type ReferralEventRepository struct {
	mu        sync.RWMutex
	byEventID map[string]domain.ReferralEvent
}

func (r *ReferralEventRepository) Create(_ context.Context, row domain.ReferralEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byEventID[row.EventID]; ok {
		return domain.ErrConflict
	}
	r.byEventID[row.EventID] = row
	return nil
}
func (r *ReferralEventRepository) GetByEventID(_ context.Context, eventID string) (domain.ReferralEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.byEventID[eventID]
	if !ok {
		return domain.ReferralEvent{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ReferralEventRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byEventID)
}

type FraudDecisionRepository struct {
	mu           sync.RWMutex
	byDecisionID map[string]domain.FraudDecision
	byEventID    map[string]string
	order        []string
}

func (r *FraudDecisionRepository) Create(_ context.Context, row domain.FraudDecision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byDecisionID[row.DecisionID]; ok {
		return domain.ErrConflict
	}
	if id, ok := r.byEventID[row.EventID]; ok && id != "" {
		return domain.ErrConflict
	}
	r.byDecisionID[row.DecisionID] = row
	r.byEventID[row.EventID] = row.DecisionID
	r.order = append(r.order, row.DecisionID)
	return nil
}
func (r *FraudDecisionRepository) GetByEventID(_ context.Context, eventID string) (domain.FraudDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEventID[eventID]
	if !ok {
		return domain.FraudDecision{}, domain.ErrNotFound
	}
	row, ok := r.byDecisionID[id]
	if !ok {
		return domain.FraudDecision{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *FraudDecisionRepository) GetByDecisionID(_ context.Context, decisionID string) (domain.FraudDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.byDecisionID[decisionID]
	if !ok {
		return domain.FraudDecision{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *FraudDecisionRepository) Update(_ context.Context, row domain.FraudDecision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byDecisionID[row.DecisionID]; !ok {
		return domain.ErrNotFound
	}
	r.byDecisionID[row.DecisionID] = row
	r.byEventID[row.EventID] = row.DecisionID
	return nil
}
func (r *FraudDecisionRepository) ListRecent(_ context.Context, limit int) ([]domain.FraudDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.FraudDecision, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.byDecisionID[r.order[i]])
	}
	return out, nil
}

type RiskPolicyRepository struct {
	mu      sync.RWMutex
	records map[string]domain.RiskPolicy
}

func (r *RiskPolicyRepository) ListActive(_ context.Context) ([]domain.RiskPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.RiskPolicy, 0, len(r.records))
	for _, row := range r.records {
		if row.IsActive {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *RiskPolicyRepository) Upsert(_ context.Context, row domain.RiskPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.PolicyID] = row
	return nil
}

type fingerprintState struct {
	id        string
	hash      string
	ips       map[string]struct{}
	seen      int
	updatedAt time.Time
}
type DeviceFingerprintRepository struct {
	mu     sync.Mutex
	byHash map[string]fingerprintState
}

func (r *DeviceFingerprintRepository) UpsertSeen(_ context.Context, fingerprintHash, ip string, at time.Time) (domain.DeviceFingerprint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.byHash[fingerprintHash]
	if st.id == "" {
		st.id = uuid.NewString()
		st.hash = fingerprintHash
		st.ips = map[string]struct{}{}
	}
	if stringsTrim(ip) != "" {
		st.ips[stringsTrim(ip)] = struct{}{}
	}
	st.seen++
	st.updatedAt = at
	r.byHash[fingerprintHash] = st
	return domain.DeviceFingerprint{DeviceFingerprintID: st.id, FingerprintHash: st.hash, LastSeenIP: stringsTrim(ip), DistinctIPCount: len(st.ips), SeenCount: st.seen, UpdatedAt: st.updatedAt}, nil
}

type clusterState struct{ row domain.Cluster }
type ClusterRepository struct {
	mu    sync.Mutex
	byKey map[string]clusterState
}

func (r *ClusterRepository) UpsertByKey(_ context.Context, key, reason string, at time.Time) (domain.Cluster, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.byKey[key]
	if st.row.ClusterID == "" {
		st.row = domain.Cluster{ClusterID: uuid.NewString(), Key: key, Reason: reason, Size: 0, CreatedAt: at, UpdatedAt: at}
	}
	st.row.Size++
	st.row.UpdatedAt = at
	r.byKey[key] = st
	return st.row, nil
}

type DisputeCaseRepository struct {
	mu           sync.RWMutex
	byDisputeID  map[string]domain.DisputeCase
	byDecisionID map[string]string
	order        []string
}

func (r *DisputeCaseRepository) Create(_ context.Context, row domain.DisputeCase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byDisputeID[row.DisputeID]; ok {
		return domain.ErrConflict
	}
	if existingID, ok := r.byDecisionID[row.DecisionID]; ok && existingID != "" {
		return domain.ErrConflict
	}
	r.byDisputeID[row.DisputeID] = row
	r.byDecisionID[row.DecisionID] = row.DisputeID
	r.order = append(r.order, row.DisputeID)
	return nil
}
func (r *DisputeCaseRepository) GetByDecisionID(_ context.Context, decisionID string) (domain.DisputeCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byDecisionID[decisionID]
	if !ok {
		return domain.DisputeCase{}, domain.ErrNotFound
	}
	row, ok := r.byDisputeID[id]
	if !ok {
		return domain.DisputeCase{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *DisputeCaseRepository) ListByStatus(_ context.Context, status string, limit int) ([]domain.DisputeCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]domain.DisputeCase, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.byDisputeID[r.order[i]]
		if status == "" || row.Status == status {
			out = append(out, row)
		}
	}
	return out, nil
}

type AuditLogRepository struct {
	mu      sync.RWMutex
	records []domain.AuditLog
}

func (r *AuditLogRepository) Create(_ context.Context, row domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, row)
	return nil
}
func (r *AuditLogRepository) ListRecent(_ context.Context, limit int) ([]domain.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit >= len(r.records) {
		out := make([]domain.AuditLog, len(r.records))
		copy(out, r.records)
		return out, nil
	}
	start := len(r.records) - limit
	out := make([]domain.AuditLog, limit)
	copy(out, r.records[start:])
	return out, nil
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
	cp := rec
	cp.ResponseBody = append([]byte(nil), rec.ResponseBody...)
	return &cp, nil
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
	rec.ResponseBody = append([]byte(nil), responseBody...)
	if at.After(rec.ExpiresAt) {
		rec.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.records[key] = rec
	return nil
}

type dedupRecord struct {
	eventType string
	expiresAt time.Time
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
	if now.After(rec.expiresAt) {
		delete(r.records, eventID)
		return false, nil
	}
	return true, nil
}
func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[eventID] = dedupRecord{eventType: eventType, expiresAt: expiresAt}
	return nil
}

type OutboxRepository struct {
	mu      sync.Mutex
	records map[string]ports.OutboxRecord
	order   []string
}

func (r *OutboxRepository) Enqueue(_ context.Context, row ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.RecordID] = row
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
		row, ok := r.records[id]
		if !ok || row.SentAt != nil {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.records[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	row.SentAt = &at
	r.records[recordID] = row
	return nil
}

func stringsTrim(v string) string { return stringTrim(v) }
func stringTrim(v string) string  { return strings.TrimSpace(v) }
