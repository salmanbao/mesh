package postgres

import (
	"context"
	"slices"
	"sync"
	"time"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
)

type Repositories struct {
	Transactions *TransactionRepository
	Refunds      *RefundRepository
	Balances     *BalanceRepository
	Webhooks     *WebhookRepository
	Idempotency  *IdempotencyRepository
	EventDedup   *EventDedupRepository
	Outbox       *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Transactions: &TransactionRepository{
			records:           make(map[string]domain.Transaction),
			byProviderTxnID:   make(map[string]string),
			insertionOrdering: []string{},
		},
		Refunds: &RefundRepository{
			records: make(map[string][]domain.Refund),
		},
		Balances: &BalanceRepository{
			records: make(map[string]domain.UserBalance),
		},
		Webhooks: &WebhookRepository{
			records: make(map[string]webhookRecord),
		},
		Idempotency: &IdempotencyRepository{
			records: make(map[string]ports.IdempotencyRecord),
		},
		EventDedup: &EventDedupRepository{
			records: make(map[string]dedupRecord),
		},
		Outbox: &OutboxRepository{
			records: make(map[string]ports.OutboxRecord),
		},
	}
}

type TransactionRepository struct {
	mu                sync.RWMutex
	records           map[string]domain.Transaction
	byProviderTxnID   map[string]string
	insertionOrdering []string
}

func (r *TransactionRepository) Create(_ context.Context, transaction domain.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[transaction.TransactionID] = transaction
	if transaction.ProviderTransactionID != "" {
		r.byProviderTxnID[transaction.ProviderTransactionID] = transaction.TransactionID
	}
	r.insertionOrdering = append(r.insertionOrdering, transaction.TransactionID)
	return nil
}

func (r *TransactionRepository) Update(_ context.Context, transaction domain.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[transaction.TransactionID]; !ok {
		return domain.ErrNotFound
	}
	r.records[transaction.TransactionID] = transaction
	if transaction.ProviderTransactionID != "" {
		r.byProviderTxnID[transaction.ProviderTransactionID] = transaction.TransactionID
	}
	return nil
}

func (r *TransactionRepository) GetByID(_ context.Context, transactionID string) (domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[transactionID]
	if !ok {
		return domain.Transaction{}, domain.ErrNotFound
	}
	return record, nil
}

func (r *TransactionRepository) GetByProviderTransactionID(_ context.Context, providerTransactionID string) (domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	transactionID, ok := r.byProviderTxnID[providerTransactionID]
	if !ok {
		return domain.Transaction{}, domain.ErrNotFound
	}
	record, ok := r.records[transactionID]
	if !ok {
		return domain.Transaction{}, domain.ErrNotFound
	}
	return record, nil
}

func (r *TransactionRepository) List(_ context.Context, query ports.TransactionListQuery) ([]domain.Transaction, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]domain.Transaction, 0, len(r.records))
	for _, record := range r.records {
		if query.UserID != "" && record.UserID != query.UserID {
			continue
		}
		items = append(items, record)
	}
	slices.SortFunc(items, func(a, b domain.Transaction) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})

	total := len(items)
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	if query.Offset >= len(items) {
		return []domain.Transaction{}, total, nil
	}
	end := query.Offset + query.Limit
	if end > len(items) {
		end = len(items)
	}
	out := make([]domain.Transaction, end-query.Offset)
	copy(out, items[query.Offset:end])
	return out, total, nil
}

type RefundRepository struct {
	mu      sync.RWMutex
	records map[string][]domain.Refund
}

func (r *RefundRepository) Create(_ context.Context, refund domain.Refund) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[refund.TransactionID] = append(r.records[refund.TransactionID], refund)
	return nil
}

func (r *RefundRepository) ListByTransaction(_ context.Context, transactionID string) ([]domain.Refund, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.records[transactionID]
	out := make([]domain.Refund, len(items))
	copy(out, items)
	return out, nil
}

type BalanceRepository struct {
	mu      sync.RWMutex
	records map[string]domain.UserBalance
}

func (r *BalanceRepository) GetOrCreate(_ context.Context, userID string) (domain.UserBalance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[userID]
	if ok {
		if strings.TrimSpace(record.Currency) == "" {
			record.Currency = "USD"
			r.records[userID] = record
		}
		return record, nil
	}
	record = domain.UserBalance{
		BalanceID:        uuid.NewString(),
		UserID:           userID,
		AvailableBalance: 0,
		PendingBalance:   0,
		ReservedBalance:  0,
		NegativeBalance:  0,
		Currency:         "USD",
		UpdatedAt:        time.Now().UTC(),
	}
	r.records[userID] = record
	return record, nil
}

func (r *BalanceRepository) Upsert(_ context.Context, balance domain.UserBalance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if strings.TrimSpace(balance.Currency) == "" {
		balance.Currency = "USD"
	}
	r.records[balance.UserID] = balance
	return nil
}

type webhookRecord struct {
	record    domain.Webhook
	expiresAt time.Time
}

type WebhookRepository struct {
	mu      sync.Mutex
	records map[string]webhookRecord
}

func (r *WebhookRepository) IsDuplicate(_ context.Context, webhookID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[webhookID]
	if !ok {
		return false, nil
	}
	if now.After(record.expiresAt) {
		delete(r.records, webhookID)
		return false, nil
	}
	return true, nil
}

func (r *WebhookRepository) MarkProcessed(_ context.Context, record domain.Webhook, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[record.WebhookID] = webhookRecord{record: record, expiresAt: expiresAt}
	return nil
}

type IdempotencyRepository struct {
	mu      sync.Mutex
	records map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[key]
	if !ok {
		return nil, nil
	}
	if now.After(record.ExpiresAt) {
		delete(r.records, key)
		return nil, nil
	}
	clone := record
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
	r.records[key] = ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   expiresAt,
	}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[key]
	if !ok {
		return domain.ErrNotFound
	}
	record.ResponseCode = responseCode
	record.ResponseBody = slices.Clone(responseBody)
	if at.After(record.ExpiresAt) {
		record.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.records[key] = record
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
	record, ok := r.records[eventID]
	if !ok {
		return false, nil
	}
	if now.After(record.ExpiresAt) {
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

func (r *OutboxRepository) Enqueue(_ context.Context, record ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[record.RecordID] = record
	r.order = append(r.order, record.RecordID)
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
		record, ok := r.records[id]
		if !ok || record.SentAt != nil {
			continue
		}
		out = append(out, record)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	record.SentAt = &at
	r.records[recordID] = record
	return nil
}
