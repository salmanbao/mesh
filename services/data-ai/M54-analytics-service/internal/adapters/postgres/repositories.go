package postgres

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/ports"
)

type Repositories struct {
	Warehouse   *WarehouseRepository
	Exports     *ExportRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Warehouse: &WarehouseRepository{
			users:        map[string]domain.DimUser{},
			campaigns:    map[string]domain.DimCampaign{},
			submissions:  map[string]domain.FactSubmission{},
			payouts:      map[string]domain.FactPayout{},
			transactions: map[string]domain.FactTransaction{},
			clicks:       map[string]domain.FactClick{},
			earnings:     map[string]domain.DailyEarnings{},
		},
		Exports:     &ExportRepository{records: map[string]domain.ExportJob{}},
		Idempotency: &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{records: map[string]dedupRecord{}},
	}
}

type WarehouseRepository struct {
	mu sync.RWMutex

	users        map[string]domain.DimUser
	campaigns    map[string]domain.DimCampaign
	submissions  map[string]domain.FactSubmission
	payouts      map[string]domain.FactPayout
	transactions map[string]domain.FactTransaction
	clicks       map[string]domain.FactClick
	earnings     map[string]domain.DailyEarnings
}

func (r *WarehouseRepository) UpsertUser(_ context.Context, row domain.DimUser) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.users[row.UserID]
	if current.CreatedAt.IsZero() {
		current.CreatedAt = row.CreatedAt
	}
	if row.Role != "" {
		current.Role = row.Role
	}
	if row.Country != "" {
		current.Country = row.Country
	}
	if row.UserID != "" {
		current.UserID = row.UserID
	}
	current.ConsentAnalytics = row.ConsentAnalytics || current.ConsentAnalytics
	if !row.UpdatedAt.IsZero() {
		current.UpdatedAt = row.UpdatedAt
	}
	r.users[row.UserID] = current
	return nil
}

func (r *WarehouseRepository) UpsertCampaign(_ context.Context, row domain.DimCampaign) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.campaigns[row.CampaignID] = row
	return nil
}

func (r *WarehouseRepository) AddSubmission(_ context.Context, row domain.FactSubmission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.submissions[row.SubmissionID] = row
	return nil
}

func (r *WarehouseRepository) AddPayout(_ context.Context, row domain.FactPayout) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payouts[row.PayoutID] = row
	return nil
}

func (r *WarehouseRepository) AddTransaction(_ context.Context, row domain.FactTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transactions[row.TransactionID] = row
	return nil
}

func (r *WarehouseRepository) AddClick(_ context.Context, row domain.FactClick) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clicks[row.ClickID] = row
	return nil
}

func (r *WarehouseRepository) UpsertDailyEarnings(_ context.Context, row domain.DailyEarnings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := row.DayDate + "::" + row.CreatorID
	current := r.earnings[key]
	current.DayDate = row.DayDate
	current.CreatorID = row.CreatorID
	current.GrossEarnings += row.GrossEarnings
	current.NetEarnings += row.NetEarnings
	current.Payouts += row.Payouts
	current.Refunds += row.Refunds
	current.UpdatedAt = row.UpdatedAt
	r.earnings[key] = current
	return nil
}

func (r *WarehouseRepository) GetCreatorDashboard(_ context.Context, userID string, from, to time.Time) (domain.CreatorDashboard, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := domain.CreatorDashboard{
		UserID:          userID,
		TopPlatforms:    map[string]int{},
		SourceBreakdown: map[string]float64{},
	}
	for _, submission := range r.submissions {
		if submission.CreatorID != userID {
			continue
		}
		if submission.OccurredAt.Before(from) || submission.OccurredAt.After(to) {
			continue
		}
		out.Submissions++
		if submission.Status == "approved" {
			out.Approved++
		}
		out.TotalViews += submission.Views
		out.TopPlatforms[submission.Platform] = out.TopPlatforms[submission.Platform] + 1
	}
	for _, payout := range r.payouts {
		if payout.CreatorID != userID {
			continue
		}
		if payout.OccurredAt.Before(from) || payout.OccurredAt.After(to) {
			continue
		}
		out.TotalPayouts += payout.Amount
		out.TotalEarnings += payout.Amount
	}
	for _, transaction := range r.transactions {
		if transaction.UserID != userID {
			continue
		}
		if transaction.OccurredAt.Before(from) || transaction.OccurredAt.After(to) {
			continue
		}
		if transaction.Refunded {
			out.TotalRefunds += transaction.Amount
			continue
		}
		out.TotalEarnings += transaction.Amount
	}
	out.DataFreshnessS = 15
	if out.TotalEarnings > 0 {
		out.SourceBreakdown["payouts"] = out.TotalPayouts / out.TotalEarnings
	}
	return out, nil
}

func (r *WarehouseRepository) GetFinancialReport(_ context.Context, from, to time.Time) (domain.FinancialReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := domain.FinancialReport{TopCreators: make([]domain.TopCreator, 0, 10)}
	creatorEarnings := map[string]float64{}
	creatorSubmissions := map[string]int{}
	creatorRefunds := map[string]float64{}

	for _, transaction := range r.transactions {
		if transaction.OccurredAt.Before(from) || transaction.OccurredAt.After(to) {
			continue
		}
		report.GMV += transaction.Amount
		if transaction.Refunded {
			creatorRefunds[transaction.UserID] += transaction.Amount
		}
		creatorEarnings[transaction.UserID] += transaction.Amount
	}
	for _, payout := range r.payouts {
		if payout.OccurredAt.Before(from) || payout.OccurredAt.After(to) {
			continue
		}
		report.TotalPayoutLiability += payout.Amount
	}
	for _, submission := range r.submissions {
		if submission.OccurredAt.Before(from) || submission.OccurredAt.After(to) {
			continue
		}
		creatorSubmissions[submission.CreatorID] = creatorSubmissions[submission.CreatorID] + 1
	}

	report.NetRevenue = report.GMV - report.TotalPayoutLiability
	totalRefunds := 0.0
	for _, amount := range creatorRefunds {
		totalRefunds += amount
	}
	if report.GMV > 0 {
		report.RefundRate = totalRefunds / report.GMV
	}

	for creatorID, earnings := range creatorEarnings {
		refundRate := 0.0
		if earnings > 0 {
			refundRate = creatorRefunds[creatorID] / earnings
		}
		report.TopCreators = append(report.TopCreators, domain.TopCreator{
			UserID:      creatorID,
			Earnings:    earnings,
			Submissions: creatorSubmissions[creatorID],
			RefundRate:  refundRate,
		})
	}
	slices.SortFunc(report.TopCreators, func(a, b domain.TopCreator) int {
		if a.Earnings == b.Earnings {
			return 0
		}
		if a.Earnings > b.Earnings {
			return -1
		}
		return 1
	})
	if len(report.TopCreators) > 10 {
		report.TopCreators = report.TopCreators[:10]
	}
	return report, nil
}

type ExportRepository struct {
	mu      sync.RWMutex
	records map[string]domain.ExportJob
}

func (r *ExportRepository) Create(_ context.Context, row domain.ExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[row.ExportID] = row
	return nil
}

func (r *ExportRepository) Update(_ context.Context, row domain.ExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.ExportID]; !ok {
		return domain.ErrNotFound
	}
	r.records[row.ExportID] = row
	return nil
}

func (r *ExportRepository) GetByID(_ context.Context, exportID string) (domain.ExportJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[exportID]
	if !ok {
		return domain.ExportJob{}, domain.ErrNotFound
	}
	return row, nil
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
