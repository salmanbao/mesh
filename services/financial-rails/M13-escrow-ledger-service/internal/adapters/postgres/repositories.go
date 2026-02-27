package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/ports"
)

type Repositories struct {
	Holds       *EscrowHoldRepository
	Ledger      *LedgerEntryRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Holds:       &EscrowHoldRepository{rows: map[string]domain.EscrowHold{}},
		Ledger:      &LedgerEntryRepository{rows: []domain.LedgerEntry{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]eventDedupRow{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type EscrowHoldRepository struct {
	mu   sync.Mutex
	rows map[string]domain.EscrowHold
}
func (r *EscrowHoldRepository) Create(_ context.Context, row domain.EscrowHold) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.rows[row.EscrowID]; ok { return domain.ErrConflict }; r.rows[row.EscrowID] = row; return nil
}
func (r *EscrowHoldRepository) GetByID(_ context.Context, escrowID string) (domain.EscrowHold, error) {
	r.mu.Lock(); defer r.mu.Unlock(); row, ok := r.rows[strings.TrimSpace(escrowID)]; if !ok { return domain.EscrowHold{}, domain.ErrNotFound }; return row, nil
}
func (r *EscrowHoldRepository) Update(_ context.Context, row domain.EscrowHold) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.rows[row.EscrowID]; !ok { return domain.ErrNotFound }; r.rows[row.EscrowID] = row; return nil
}

type LedgerEntryRepository struct {
	mu   sync.Mutex
	rows []domain.LedgerEntry
}
func (r *LedgerEntryRepository) Append(_ context.Context, row domain.LedgerEntry) error { r.mu.Lock(); defer r.mu.Unlock(); r.rows = append(r.rows, row); return nil }
func (r *LedgerEntryRepository) ListByCampaignID(_ context.Context, campaignID string) ([]domain.LedgerEntry, error) {
	r.mu.Lock(); defer r.mu.Unlock(); id := strings.TrimSpace(campaignID); out := make([]domain.LedgerEntry, 0)
	for _, row := range r.rows { if row.CampaignID == id { out = append(out, row) } }
	sort.Slice(out, func(i,j int) bool { return out[i].OccurredAt.Before(out[j].OccurredAt) })
	return out, nil
}

type IdempotencyRepository struct { mu sync.Mutex; rows map[string]ports.IdempotencyRecord }
func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock(); defer r.mu.Unlock(); row, ok := r.rows[key]; if !ok { return nil, nil }; if now.After(row.ExpiresAt) { delete(r.rows, key); return nil, nil }; c := row; c.ResponseBody = append([]byte(nil), row.ResponseBody...); return &c, nil
}
func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock(); if row, ok := r.rows[key]; ok && time.Now().UTC().Before(row.ExpiresAt) { return domain.ErrConflict }; r.rows[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}; return nil
}
func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, _ time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock(); row, ok := r.rows[key]; if !ok { return domain.ErrNotFound }; row.ResponseCode = responseCode; row.ResponseBody = append([]byte(nil), responseBody...); r.rows[key] = row; return nil
}

type eventDedupRow struct { EventID string; EventType string; ExpiresAt time.Time }
type EventDedupRepository struct { mu sync.Mutex; rows map[string]eventDedupRow }
func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock(); defer r.mu.Unlock(); row, ok := r.rows[eventID]; if !ok { return false, nil }; if now.After(row.ExpiresAt) { delete(r.rows, eventID); return false, nil }; return true, nil
}
func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error { r.mu.Lock(); defer r.mu.Unlock(); r.rows[eventID] = eventDedupRow{EventID: eventID, EventType: eventType, ExpiresAt: expiresAt}; return nil }

type OutboxRepository struct { mu sync.Mutex; rows map[string]ports.OutboxRecord; order []string }
func (r *OutboxRepository) Enqueue(_ context.Context, row ports.OutboxRecord) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.rows[row.RecordID]; ok { return domain.ErrConflict }; r.rows[row.RecordID] = row; r.order = append(r.order, row.RecordID); return nil }
func (r *OutboxRepository) ListPending(_ context.Context, limit int) ([]ports.OutboxRecord, error) {
	r.mu.Lock(); defer r.mu.Unlock(); if limit <= 0 { limit = 100 }; out := make([]ports.OutboxRecord, 0, limit)
	for _, id := range r.order { row, ok := r.rows[id]; if !ok || row.SentAt != nil { continue }; out = append(out, row); if len(out) >= limit { break } }
	return out, nil
}
func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error { r.mu.Lock(); defer r.mu.Unlock(); row, ok := r.rows[recordID]; if !ok { return domain.ErrNotFound }; row.SentAt = &at; r.rows[recordID] = row; return nil }
