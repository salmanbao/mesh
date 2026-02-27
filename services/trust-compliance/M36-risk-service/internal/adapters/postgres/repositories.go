package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/ports"
)

type Repositories struct {
	RiskProfiles *SellerRiskProfileRepository
	Escrow       *SellerEscrowRepository
	Disputes     *DisputeLogRepository
	Evidence     *DisputeEvidenceRepository
	FraudFlags   *FraudPatternFlagRepository
	ReserveLogs  *ReserveTriggerLogRepository
	DebtLogs     *SellerDebtLogRepository
	Suspensions  *SellerSuspensionLogRepository
	Idempotency  *IdempotencyRepository
	EventDedup   *EventDedupRepository
	Outbox       *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		RiskProfiles: &SellerRiskProfileRepository{records: map[string]domain.SellerRiskProfile{}},
		Escrow:       &SellerEscrowRepository{records: map[string]domain.SellerEscrow{}},
		Disputes:     &DisputeLogRepository{records: map[string]domain.DisputeLog{}, byTxn: map[string]string{}},
		Evidence:     &DisputeEvidenceRepository{records: map[string][]domain.DisputeEvidence{}},
		FraudFlags:   &FraudPatternFlagRepository{records: map[string][]domain.FraudPatternFlag{}},
		ReserveLogs:  &ReserveTriggerLogRepository{records: map[string][]domain.ReserveTriggerLog{}},
		DebtLogs:     &SellerDebtLogRepository{records: map[string][]domain.SellerDebtLog{}},
		Suspensions:  &SellerSuspensionLogRepository{records: map[string][]domain.SellerSuspensionLog{}},
		Idempotency:  &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:   &EventDedupRepository{records: map[string]dedupRecord{}},
		Outbox:       &OutboxRepository{records: map[string]ports.OutboxRecord{}},
	}
}

type SellerRiskProfileRepository struct {
	mu      sync.RWMutex
	records map[string]domain.SellerRiskProfile
}

func (r *SellerRiskProfileRepository) Upsert(_ context.Context, row domain.SellerRiskProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.SellerID] = row
	return nil
}

func (r *SellerRiskProfileRepository) GetBySellerID(_ context.Context, sellerID string) (domain.SellerRiskProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[sellerID]
	if !ok {
		return domain.SellerRiskProfile{}, domain.ErrNotFound
	}
	return row, nil
}

type SellerEscrowRepository struct {
	mu      sync.RWMutex
	records map[string]domain.SellerEscrow
}

func (r *SellerEscrowRepository) Upsert(_ context.Context, row domain.SellerEscrow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.SellerID] = row
	return nil
}

func (r *SellerEscrowRepository) GetBySellerID(_ context.Context, sellerID string) (domain.SellerEscrow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[sellerID]
	if !ok {
		return domain.SellerEscrow{}, domain.ErrNotFound
	}
	return row, nil
}

type DisputeLogRepository struct {
	mu      sync.RWMutex
	records map[string]domain.DisputeLog
	byTxn   map[string]string
	order   []string
}

func (r *DisputeLogRepository) Create(_ context.Context, row domain.DisputeLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.DisputeID]; ok {
		return domain.ErrConflict
	}
	if row.TransactionID != "" {
		if existingID, ok := r.byTxn[row.TransactionID]; ok {
			if existing, ok := r.records[existingID]; ok && existing.DisputeID != "" {
				return domain.ErrConflict
			}
		}
		r.byTxn[row.TransactionID] = row.DisputeID
	}
	r.records[row.DisputeID] = row
	r.order = append(r.order, row.DisputeID)
	return nil
}

func (r *DisputeLogRepository) Update(_ context.Context, row domain.DisputeLog) error {
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

func (r *DisputeLogRepository) GetByID(_ context.Context, disputeID string) (domain.DisputeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[disputeID]
	if !ok {
		return domain.DisputeLog{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *DisputeLogRepository) GetByTransactionID(_ context.Context, transactionID string) (domain.DisputeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byTxn[transactionID]
	if !ok {
		return domain.DisputeLog{}, domain.ErrNotFound
	}
	row, ok := r.records[id]
	if !ok {
		return domain.DisputeLog{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *DisputeLogRepository) ListBySeller(_ context.Context, sellerID string, limit int) ([]domain.DisputeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	out := make([]domain.DisputeLog, 0, limit)
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		row := r.records[r.order[i]]
		if row.SellerID == sellerID {
			out = append(out, row)
		}
	}
	return out, nil
}

type DisputeEvidenceRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.DisputeEvidence
}

func (r *DisputeEvidenceRepository) Create(_ context.Context, row domain.DisputeEvidence) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.DisputeID] = append(r.records[row.DisputeID], row)
	return nil
}

func (r *DisputeEvidenceRepository) ListByDispute(_ context.Context, disputeID string) ([]domain.DisputeEvidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.records[disputeID]
	out := make([]domain.DisputeEvidence, len(items))
	copy(out, items)
	return out, nil
}

type FraudPatternFlagRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.FraudPatternFlag
}

func (r *FraudPatternFlagRepository) Create(_ context.Context, row domain.FraudPatternFlag) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.SellerID] = append(r.records[row.SellerID], row)
	return nil
}

func (r *FraudPatternFlagRepository) ListBySeller(_ context.Context, sellerID string, limit int) ([]domain.FraudPatternFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneFraudFlags(r.records[sellerID], limit), nil
}

type ReserveTriggerLogRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.ReserveTriggerLog
}

func (r *ReserveTriggerLogRepository) Create(_ context.Context, row domain.ReserveTriggerLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.SellerID] = append(r.records[row.SellerID], row)
	return nil
}

func (r *ReserveTriggerLogRepository) ListBySeller(_ context.Context, sellerID string, limit int) ([]domain.ReserveTriggerLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneReserveLogs(r.records[sellerID], limit), nil
}

type SellerDebtLogRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.SellerDebtLog
}

func (r *SellerDebtLogRepository) Create(_ context.Context, row domain.SellerDebtLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.SellerID] = append(r.records[row.SellerID], row)
	return nil
}

func (r *SellerDebtLogRepository) ListBySeller(_ context.Context, sellerID string, limit int) ([]domain.SellerDebtLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneDebtLogs(r.records[sellerID], limit), nil
}

type SellerSuspensionLogRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.SellerSuspensionLog
}

func (r *SellerSuspensionLogRepository) Create(_ context.Context, row domain.SellerSuspensionLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.SellerID] = append(r.records[row.SellerID], row)
	return nil
}

func (r *SellerSuspensionLogRepository) ListBySeller(_ context.Context, sellerID string, limit int) ([]domain.SellerSuspensionLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneSuspensions(r.records[sellerID], limit), nil
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
	clone.ResponseBody = append([]byte(nil), rec.ResponseBody...)
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
	rec.ResponseBody = append([]byte(nil), responseBody...)
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

func cloneFraudFlags(items []domain.FraudPatternFlag, limit int) []domain.FraudPatternFlag {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]domain.FraudPatternFlag, 0, limit)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, items[i])
	}
	return out
}

func cloneReserveLogs(items []domain.ReserveTriggerLog, limit int) []domain.ReserveTriggerLog {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]domain.ReserveTriggerLog, 0, limit)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, items[i])
	}
	return out
}

func cloneDebtLogs(items []domain.SellerDebtLog, limit int) []domain.SellerDebtLog {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]domain.SellerDebtLog, 0, limit)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, items[i])
	}
	return out
}

func cloneSuspensions(items []domain.SellerSuspensionLog, limit int) []domain.SellerSuspensionLog {
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	out := make([]domain.SellerSuspensionLog, 0, limit)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, items[i])
	}
	return out
}
