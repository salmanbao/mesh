package postgres

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
)

type Repositories struct {
	Invoices    *InvoiceRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Invoices: &InvoiceRepository{
			invoices:      make(map[string]domain.Invoice),
			dailySequence: make(map[string]int),
		},
		Idempotency: &IdempotencyRepository{
			records: make(map[string]ports.IdempotencyRecord),
		},
		EventDedup: &EventDedupRepository{
			records: make(map[string]dedupRecord),
		},
	}
}

type InvoiceRepository struct {
	mu            sync.RWMutex
	invoices      map[string]domain.Invoice
	dailySequence map[string]int
	emailEvents   []domain.InvoiceEmailEvent
	voidHistory   []domain.VoidHistory
	payments      []domain.InvoicePayment
	payouts       []domain.PayoutReceipt
}

func (r *InvoiceRepository) NextInvoiceSequence(_ context.Context, day time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := day.UTC().Format("2006-01-02")
	r.dailySequence[key]++
	return r.dailySequence[key], nil
}

func (r *InvoiceRepository) Create(_ context.Context, invoice domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invoices[invoice.InvoiceID] = invoice
	return nil
}

func (r *InvoiceRepository) GetByID(_ context.Context, invoiceID string) (domain.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	invoice, ok := r.invoices[invoiceID]
	if !ok {
		return domain.Invoice{}, domain.ErrNotFound
	}
	return invoice, nil
}

func (r *InvoiceRepository) Update(_ context.Context, invoice domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.invoices[invoice.InvoiceID]; !ok {
		return domain.ErrNotFound
	}
	r.invoices[invoice.InvoiceID] = invoice
	return nil
}

func (r *InvoiceRepository) ListByCustomer(_ context.Context, customerID string, query ports.InvoiceQuery) ([]domain.Invoice, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	filtered := make([]domain.Invoice, 0)
	for _, invoice := range r.invoices {
		if invoice.CustomerID != customerID {
			continue
		}
		if !matchesInvoiceQuery(invoice, query) {
			continue
		}
		filtered = append(filtered, invoice)
	}
	slices.SortFunc(filtered, func(a, b domain.Invoice) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	total := len(filtered)
	return paginate(filtered, query.Limit, query.Offset), total, nil
}

func (r *InvoiceRepository) Search(_ context.Context, query ports.InvoiceQuery) ([]domain.Invoice, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	filtered := make([]domain.Invoice, 0)
	for _, invoice := range r.invoices {
		if !matchesInvoiceQuery(invoice, query) {
			continue
		}
		filtered = append(filtered, invoice)
	}
	slices.SortFunc(filtered, func(a, b domain.Invoice) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	total := len(filtered)
	return paginate(filtered, query.Limit, query.Offset), total, nil
}

func (r *InvoiceRepository) RecordEmailEvent(_ context.Context, event domain.InvoiceEmailEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emailEvents = append(r.emailEvents, event)
	return nil
}

func (r *InvoiceRepository) RecordVoid(_ context.Context, record domain.VoidHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.voidHistory = append(r.voidHistory, record)
	return nil
}

func (r *InvoiceRepository) RecordPayment(_ context.Context, payment domain.InvoicePayment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payments = append(r.payments, payment)
	return nil
}

func (r *InvoiceRepository) CreatePayoutReceipt(_ context.Context, receipt domain.PayoutReceipt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payouts = append(r.payouts, receipt)
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

func paginate(invoices []domain.Invoice, limit, offset int) []domain.Invoice {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(invoices) {
		return []domain.Invoice{}
	}
	end := offset + limit
	if end > len(invoices) {
		end = len(invoices)
	}
	out := make([]domain.Invoice, end-offset)
	copy(out, invoices[offset:end])
	return out
}

func matchesInvoiceQuery(invoice domain.Invoice, query ports.InvoiceQuery) bool {
	if query.Status != "" && string(invoice.Status) != query.Status {
		return false
	}
	if query.InvoiceNumber != "" && !strings.Contains(strings.ToLower(invoice.InvoiceNumber), strings.ToLower(query.InvoiceNumber)) {
		return false
	}
	if query.CustomerEmail != "" && !strings.Contains(strings.ToLower(invoice.CustomerEmail), strings.ToLower(query.CustomerEmail)) {
		return false
	}
	if query.MinAmount > 0 && invoice.Total < query.MinAmount {
		return false
	}
	if query.MaxAmount > 0 && invoice.Total > query.MaxAmount {
		return false
	}
	if query.DateFrom != nil && invoice.InvoiceDate.Before(*query.DateFrom) {
		return false
	}
	if query.DateTo != nil && invoice.InvoiceDate.After(*query.DateTo) {
		return false
	}
	return true
}
