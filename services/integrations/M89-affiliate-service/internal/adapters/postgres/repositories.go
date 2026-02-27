package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/ports"
)

type Repositories struct {
	Affiliates   *AffiliateRepository
	Links        *ReferralLinkRepository
	Clicks       *ReferralClickRepository
	Attributions *ReferralAttributionRepository
	Earnings     *AffiliateEarningRepository
	Payouts      *AffiliatePayoutRepository
	AuditLogs    *AffiliateAuditLogRepository
	Idempotency  *IdempotencyRepository
	EventDedup   *EventDedupRepository
	Outbox       *OutboxRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Affiliates:   &AffiliateRepository{byID: map[string]domain.Affiliate{}, byUserID: map[string]string{}},
		Links:        &ReferralLinkRepository{byID: map[string]domain.ReferralLink{}, byToken: map[string]string{}},
		Clicks:       &ReferralClickRepository{byID: map[string]domain.ReferralClick{}, byAffiliate: map[string][]string{}},
		Attributions: &ReferralAttributionRepository{byID: map[string]domain.ReferralAttribution{}, byOrderID: map[string]string{}, byAffiliate: map[string][]string{}},
		Earnings:     &AffiliateEarningRepository{byID: map[string]domain.AffiliateEarning{}, byAffiliate: map[string][]string{}},
		Payouts:      &AffiliatePayoutRepository{byID: map[string]domain.AffiliatePayout{}, byAffiliate: map[string][]string{}},
		AuditLogs:    &AffiliateAuditLogRepository{rows: []domain.AffiliateAuditLog{}},
		Idempotency:  &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:   &EventDedupRepository{rows: map[string]time.Time{}},
		Outbox:       &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

type AffiliateRepository struct {
	mu       sync.Mutex
	byID     map[string]domain.Affiliate
	byUserID map[string]string
}

func (r *AffiliateRepository) Create(_ context.Context, row domain.Affiliate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.AffiliateID]; ok {
		return domain.ErrConflict
	}
	if _, ok := r.byUserID[row.UserID]; ok {
		return domain.ErrConflict
	}
	r.byID[row.AffiliateID] = row
	r.byUserID[row.UserID] = row.AffiliateID
	return nil
}
func (r *AffiliateRepository) GetByID(_ context.Context, affiliateID string) (domain.Affiliate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(affiliateID)]
	if !ok {
		return domain.Affiliate{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *AffiliateRepository) GetByUserID(_ context.Context, userID string) (domain.Affiliate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byUserID[strings.TrimSpace(userID)]
	if !ok {
		return domain.Affiliate{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok {
		return domain.Affiliate{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *AffiliateRepository) Update(_ context.Context, row domain.Affiliate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.AffiliateID]; !ok {
		return domain.ErrNotFound
	}
	r.byID[row.AffiliateID] = row
	r.byUserID[row.UserID] = row.AffiliateID
	return nil
}

type ReferralLinkRepository struct {
	mu      sync.Mutex
	byID    map[string]domain.ReferralLink
	byToken map[string]string
}

func (r *ReferralLinkRepository) Create(_ context.Context, row domain.ReferralLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.LinkID]; ok {
		return domain.ErrConflict
	}
	if _, ok := r.byToken[row.Token]; ok {
		return domain.ErrConflict
	}
	r.byID[row.LinkID] = row
	r.byToken[row.Token] = row.LinkID
	return nil
}
func (r *ReferralLinkRepository) GetByID(_ context.Context, linkID string) (domain.ReferralLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(linkID)]
	if !ok {
		return domain.ReferralLink{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ReferralLinkRepository) GetByToken(_ context.Context, token string) (domain.ReferralLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byToken[strings.TrimSpace(token)]
	if !ok {
		return domain.ReferralLink{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok {
		return domain.ReferralLink{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ReferralLinkRepository) ListByAffiliateID(_ context.Context, affiliateID string) ([]domain.ReferralLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.ReferralLink{}
	for _, row := range r.byID {
		if row.AffiliateID == affiliateID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

type ReferralClickRepository struct {
	mu          sync.Mutex
	byID        map[string]domain.ReferralClick
	byAffiliate map[string][]string
}

func (r *ReferralClickRepository) Append(_ context.Context, row domain.ReferralClick) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.ClickID]; ok {
		return domain.ErrConflict
	}
	r.byID[row.ClickID] = row
	r.byAffiliate[row.AffiliateID] = append(r.byAffiliate[row.AffiliateID], row.ClickID)
	return nil
}
func (r *ReferralClickRepository) GetByID(_ context.Context, clickID string) (domain.ReferralClick, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(clickID)]
	if !ok {
		return domain.ReferralClick{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ReferralClickRepository) ListByAffiliateID(_ context.Context, affiliateID string) ([]domain.ReferralClick, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.byAffiliate[affiliateID]
	out := make([]domain.ReferralClick, 0, len(ids))
	for _, id := range ids {
		if row, ok := r.byID[id]; ok {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ClickedAt.Before(out[j].ClickedAt) })
	return out, nil
}
func (r *ReferralClickRepository) CountByLinkID(_ context.Context, linkID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, row := range r.byID {
		if row.LinkID == linkID {
			count++
		}
	}
	return count, nil
}

type ReferralAttributionRepository struct {
	mu          sync.Mutex
	byID        map[string]domain.ReferralAttribution
	byOrderID   map[string]string
	byAffiliate map[string][]string
}

func (r *ReferralAttributionRepository) Create(_ context.Context, row domain.ReferralAttribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.AttributionID]; ok {
		return domain.ErrConflict
	}
	if _, ok := r.byOrderID[row.OrderID]; ok {
		return domain.ErrConflict
	}
	r.byID[row.AttributionID] = row
	r.byOrderID[row.OrderID] = row.AttributionID
	r.byAffiliate[row.AffiliateID] = append(r.byAffiliate[row.AffiliateID], row.AttributionID)
	return nil
}
func (r *ReferralAttributionRepository) GetByOrderID(_ context.Context, orderID string) (domain.ReferralAttribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byOrderID[strings.TrimSpace(orderID)]
	if !ok {
		return domain.ReferralAttribution{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok {
		return domain.ReferralAttribution{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *ReferralAttributionRepository) ListByAffiliateID(_ context.Context, affiliateID string) ([]domain.ReferralAttribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.byAffiliate[affiliateID]
	out := make([]domain.ReferralAttribution, 0, len(ids))
	for _, id := range ids {
		if row, ok := r.byID[id]; ok {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AttributedAt.Before(out[j].AttributedAt) })
	return out, nil
}

type AffiliateEarningRepository struct {
	mu          sync.Mutex
	byID        map[string]domain.AffiliateEarning
	byAffiliate map[string][]string
}

func (r *AffiliateEarningRepository) Create(_ context.Context, row domain.AffiliateEarning) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.EarningID]; ok {
		return domain.ErrConflict
	}
	r.byID[row.EarningID] = row
	r.byAffiliate[row.AffiliateID] = append(r.byAffiliate[row.AffiliateID], row.EarningID)
	return nil
}
func (r *AffiliateEarningRepository) ListByAffiliateID(_ context.Context, affiliateID string) ([]domain.AffiliateEarning, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.byAffiliate[affiliateID]
	out := make([]domain.AffiliateEarning, 0, len(ids))
	for _, id := range ids {
		if row, ok := r.byID[id]; ok {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func (r *AffiliateEarningRepository) SumByAffiliateAndStatus(_ context.Context, affiliateID, status string) (float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0.0
	for _, id := range r.byAffiliate[affiliateID] {
		if row, ok := r.byID[id]; ok && row.Status == status {
			total += row.Amount
		}
	}
	return total, nil
}

type AffiliatePayoutRepository struct {
	mu          sync.Mutex
	byID        map[string]domain.AffiliatePayout
	byAffiliate map[string][]string
}

func (r *AffiliatePayoutRepository) Create(_ context.Context, row domain.AffiliatePayout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.PayoutID]; ok {
		return domain.ErrConflict
	}
	r.byID[row.PayoutID] = row
	r.byAffiliate[row.AffiliateID] = append(r.byAffiliate[row.AffiliateID], row.PayoutID)
	return nil
}
func (r *AffiliatePayoutRepository) ListByAffiliateID(_ context.Context, affiliateID string) ([]domain.AffiliatePayout, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.byAffiliate[affiliateID]
	out := make([]domain.AffiliatePayout, 0, len(ids))
	for _, id := range ids {
		if row, ok := r.byID[id]; ok {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].QueuedAt.After(out[j].QueuedAt) })
	return out, nil
}

type AffiliateAuditLogRepository struct {
	mu   sync.Mutex
	rows []domain.AffiliateAuditLog
}

func (r *AffiliateAuditLogRepository) Append(_ context.Context, row domain.AffiliateAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *AffiliateAuditLogRepository) ListByAffiliateID(_ context.Context, affiliateID string) ([]domain.AffiliateAuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.AffiliateAuditLog{}
	for _, row := range r.rows {
		if row.AffiliateID == affiliateID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
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
		row, ok := r.rows[id]
		if !ok || row.SentAt != nil {
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
