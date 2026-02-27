package postgres

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/ports"
)

type Repositories struct {
	Disputes     *DisputeRepository
	Messages     *MessageRepository
	Evidence     *EvidenceRepository
	Approvals    *ApprovalRepository
	AuditLogs    *AuditLogRepository
	StateHistory *StateHistoryRepository
	Mediation    *MediationRepository
	Rules        *AutoResolutionRuleRepository
	Idempotency  *IdempotencyRepository
	EventDedup   *EventDedupRepository
	Outbox       *OutboxRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	return &Repositories{
		Disputes:     &DisputeRepository{records: map[string]domain.Dispute{}, byTxn: map[string]string{}},
		Messages:     &MessageRepository{records: map[string][]domain.DisputeMessage{}},
		Evidence:     &EvidenceRepository{records: map[string][]domain.DisputeEvidence{}},
		Approvals:    &ApprovalRepository{records: map[string][]domain.DisputeApproval{}},
		AuditLogs:    &AuditLogRepository{records: map[string][]domain.DisputeAuditLog{}, global: []domain.DisputeAuditLog{}},
		StateHistory: &StateHistoryRepository{records: map[string][]domain.DisputeStateHistory{}},
		Mediation:    &MediationRepository{records: map[string]domain.DisputeMediation{}},
		Rules:        &AutoResolutionRuleRepository{records: []domain.AutoResolutionRule{{RuleID: "rule-duplicate-charge", Name: "Duplicate charge auto-resolve", Enabled: true, CreatedAt: now, UpdatedAt: now}}},
		Idempotency:  &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:   &EventDedupRepository{records: map[string]dedupRecord{}},
		Outbox:       &OutboxRepository{records: map[string]ports.OutboxRecord{}},
	}
}

type DisputeRepository struct {
	mu      sync.RWMutex
	records map[string]domain.Dispute
	byTxn   map[string]string
	order   []string
}

func (r *DisputeRepository) Create(_ context.Context, row domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.DisputeID]; ok {
		return domain.ErrConflict
	}
	if txID := row.TransactionID; txID != "" {
		if existingID, ok := r.byTxn[txID]; ok {
			if existing, ok := r.records[existingID]; ok && existing.Status != domain.DisputeStatusResolved && existing.Status != domain.DisputeStatusWithdrawn {
				return domain.ErrConflict
			}
		}
		r.byTxn[txID] = row.DisputeID
	}
	r.records[row.DisputeID] = row
	r.order = append(r.order, row.DisputeID)
	return nil
}

func (r *DisputeRepository) Update(_ context.Context, row domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.DisputeID]; !ok {
		return domain.ErrNotFound
	}
	r.records[row.DisputeID] = row
	if row.TransactionID != "" {
		r.byTxn[row.TransactionID] = row.DisputeID
	}
	return nil
}

func (r *DisputeRepository) GetByID(_ context.Context, disputeID string) (domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[disputeID]
	if !ok {
		return domain.Dispute{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *DisputeRepository) GetOpenByTransactionID(_ context.Context, transactionID string) (domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byTxn[transactionID]
	if !ok {
		return domain.Dispute{}, domain.ErrNotFound
	}
	row, ok := r.records[id]
	if !ok {
		return domain.Dispute{}, domain.ErrNotFound
	}
	if row.Status == domain.DisputeStatusResolved || row.Status == domain.DisputeStatusWithdrawn {
		return domain.Dispute{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *DisputeRepository) ListByUser(_ context.Context, userID string, limit int) ([]domain.Dispute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := make([]domain.Dispute, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.records[r.order[i]]
		if row.UserID != userID {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

type MessageRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.DisputeMessage
}

func (r *MessageRepository) Create(_ context.Context, row domain.DisputeMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.DisputeID] = append(r.records[row.DisputeID], row)
	return nil
}

func (r *MessageRepository) ListByDispute(_ context.Context, disputeID string, limit int) ([]domain.DisputeMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.records[disputeID]
	if limit <= 0 || limit >= len(items) {
		return slices.Clone(items), nil
	}
	start := len(items) - limit
	if start < 0 {
		start = 0
	}
	return slices.Clone(items[start:]), nil
}

type EvidenceRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.DisputeEvidence
}

func (r *EvidenceRepository) CreateMany(_ context.Context, rows []domain.DisputeEvidence) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range rows {
		r.records[row.DisputeID] = append(r.records[row.DisputeID], row)
	}
	return nil
}

func (r *EvidenceRepository) ListByDispute(_ context.Context, disputeID string) ([]domain.DisputeEvidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.records[disputeID]), nil
}

type ApprovalRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.DisputeApproval
}

func (r *ApprovalRepository) Create(_ context.Context, row domain.DisputeApproval) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.records[row.DisputeID] {
		if existing.ApprovalLevel == row.ApprovalLevel && existing.Status == "approved" {
			return domain.ErrConflict
		}
	}
	r.records[row.DisputeID] = append(r.records[row.DisputeID], row)
	return nil
}

func (r *ApprovalRepository) ListByDispute(_ context.Context, disputeID string) ([]domain.DisputeApproval, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.records[disputeID]), nil
}

type AuditLogRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.DisputeAuditLog
	global  []domain.DisputeAuditLog
}

func (r *AuditLogRepository) Create(_ context.Context, row domain.DisputeAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row.DisputeID != "" {
		r.records[row.DisputeID] = append(r.records[row.DisputeID], row)
	}
	r.global = append(r.global, row)
	return nil
}

func (r *AuditLogRepository) ListByDispute(_ context.Context, disputeID string, limit int) ([]domain.DisputeAuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.records[disputeID]
	if limit <= 0 || limit >= len(items) {
		return slices.Clone(items), nil
	}
	start := len(items) - limit
	if start < 0 {
		start = 0
	}
	return slices.Clone(items[start:]), nil
}

type StateHistoryRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.DisputeStateHistory
}

func (r *StateHistoryRepository) Create(_ context.Context, row domain.DisputeStateHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.DisputeID] = append(r.records[row.DisputeID], row)
	return nil
}

func (r *StateHistoryRepository) ListByDispute(_ context.Context, disputeID string) ([]domain.DisputeStateHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.records[disputeID]), nil
}

type MediationRepository struct {
	mu      sync.RWMutex
	records map[string]domain.DisputeMediation
}

func (r *MediationRepository) Upsert(_ context.Context, row domain.DisputeMediation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.DisputeID] = row
	return nil
}

type AutoResolutionRuleRepository struct {
	mu      sync.RWMutex
	records []domain.AutoResolutionRule
}

func (r *AutoResolutionRuleRepository) ListEnabled(_ context.Context) ([]domain.AutoResolutionRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.AutoResolutionRule, 0, len(r.records))
	for _, row := range r.records {
		if row.Enabled {
			out = append(out, row)
		}
	}
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
	// deterministic by create time fallback record order
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
