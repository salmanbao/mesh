package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/ports"
)

type Repositories struct {
	Warehouse   *WarehouseRepository
	Exports     *ExportJobRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	w := &WarehouseRepository{
		referralEvents:   map[string]domain.ReferralEvent{},
		dailyAggregates:  map[string]domain.ReferralAggregateDaily{},
		funnelAggregates: map[string]domain.ReferralFunnelAggregate{},
		cohorts:          map[string]domain.ReferralCohortRetention{},
		geoAggregates:    map[string]domain.ReferralGeoAggregate{},
	}
	seedWarehouse(w, now)
	return &Repositories{
		Warehouse:   w,
		Exports:     &ExportJobRepository{records: map[string]domain.ReferralExportJob{}},
		Idempotency: &IdempotencyRepository{records: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{records: map[string]dedupRecord{}},
		Outbox:      &OutboxRepository{records: map[string]ports.OutboxRecord{}},
	}
}

type WarehouseRepository struct {
	mu               sync.RWMutex
	referralEvents   map[string]domain.ReferralEvent
	dailyAggregates  map[string]domain.ReferralAggregateDaily
	funnelAggregates map[string]domain.ReferralFunnelAggregate
	cohorts          map[string]domain.ReferralCohortRetention
	geoAggregates    map[string]domain.ReferralGeoAggregate
}

func (r *WarehouseRepository) CreateReferralEvent(_ context.Context, row domain.ReferralEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	for _, existing := range r.referralEvents {
		if existing.EventID == row.EventID && existing.EventID != "" {
			return domain.ErrConflict
		}
	}
	r.referralEvents[row.ID] = row
	return nil
}
func (r *WarehouseRepository) UpsertDailyAggregate(_ context.Context, row domain.ReferralAggregateDaily) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dailyAggregates[row.ID] = row
	return nil
}
func (r *WarehouseRepository) UpsertFunnelAggregate(_ context.Context, row domain.ReferralFunnelAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funnelAggregates[row.ID] = row
	return nil
}
func (r *WarehouseRepository) UpsertCohortRetention(_ context.Context, row domain.ReferralCohortRetention) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cohorts[row.ID] = row
	return nil
}
func (r *WarehouseRepository) UpsertGeoAggregate(_ context.Context, row domain.ReferralGeoAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.geoAggregates[row.ID] = row
	return nil
}

func (r *WarehouseRepository) GetFunnel(_ context.Context, from, to time.Time) (domain.FunnelReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := domain.FunnelReport{}
	for _, row := range r.funnelAggregates {
		day, err := time.Parse("2006-01-02", row.Date)
		if err != nil {
			continue
		}
		if day.Before(dateOnly(from)) || day.After(dateOnly(to)) {
			continue
		}
		out.Clicks += row.Clicks
		out.Signups += row.Signups
		out.FirstPurchases += row.FirstPurchases
		out.RepeatPurchases += row.RepeatPurchases
	}
	out.ClickToSignupRate = domain.SafeRate(out.Signups, out.Clicks)
	out.SignupToFirstRate = domain.SafeRate(out.FirstPurchases, out.Signups)
	out.FirstToRepeatRate = domain.SafeRate(out.RepeatPurchases, out.FirstPurchases)
	return out, nil
}

func (r *WarehouseRepository) GetLeaderboard(_ context.Context, period string, now time.Time) (domain.LeaderboardReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	from := periodStart(period, now)
	type agg struct {
		clicks, conversions int
		revenue             float64
	}
	byToken := map[string]agg{}
	for _, row := range r.dailyAggregates {
		day, err := time.Parse("2006-01-02", row.Date)
		if err != nil || day.Before(dateOnly(from)) {
			continue
		}
		key := row.ReferralToken
		if key == "" {
			key = "unattributed"
		}
		a := byToken[key]
		a.clicks += row.Clicks
		a.conversions += row.Conversions
		a.revenue += row.Revenue
		byToken[key] = a
	}
	entries := make([]domain.LeaderboardEntry, 0, len(byToken))
	for token, a := range byToken {
		entries = append(entries, domain.LeaderboardEntry{ReferralToken: token, Sales: a.revenue, Conversions: a.conversions, Clicks: a.clicks, ConversionRate: domain.SafeRate(a.conversions, a.clicks), LTV90: ltvFromRevenue(a.revenue, a.conversions)})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Sales == entries[j].Sales {
			return entries[i].Conversions > entries[j].Conversions
		}
		return entries[i].Sales > entries[j].Sales
	})
	if len(entries) > 100 {
		entries = entries[:100]
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}
	return domain.LeaderboardReport{Period: period, TopPerformers: entries, GeneratedAt: now}, nil
}

func (r *WarehouseRepository) GetCohortRetention(_ context.Context, from, to time.Time) (domain.CohortRetentionReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.ReferralCohortRetention, 0)
	for _, row := range r.cohorts {
		day, err := time.Parse("2006-01-02", row.CohortDate)
		if err != nil || day.Before(dateOnly(from)) || day.After(dateOnly(to)) {
			continue
		}
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CohortDate < items[j].CohortDate })
	return domain.CohortRetentionReport{Cohorts: items}, nil
}

func (r *WarehouseRepository) GetGeo(_ context.Context, from, to time.Time) (domain.GeoPerformanceReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.ReferralGeoAggregate, 0)
	for _, row := range r.geoAggregates {
		day, err := time.Parse("2006-01-02", row.Date)
		if err != nil || day.Before(dateOnly(from)) || day.After(dateOnly(to)) {
			continue
		}
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Revenue > items[j].Revenue })
	if len(items) > 10 {
		items = items[:10]
	}
	return domain.GeoPerformanceReport{TopCountries: items}, nil
}

func (r *WarehouseRepository) GetPayoutForecast(_ context.Context, period string, now time.Time) (domain.PayoutForecast, error) {
	report, err := r.GetLeaderboard(context.Background(), period, now)
	if err != nil {
		return domain.PayoutForecast{}, err
	}
	total := 0.0
	for _, e := range report.TopPerformers {
		total += e.Sales * 0.12
	}
	return domain.PayoutForecast{Period: period, ForecastedAmount: round2(total), ConfidenceLow: round2(total * 0.9), ConfidenceHigh: round2(total * 1.1), DeviationAlert: total > 50000, GeneratedAt: now}, nil
}

type ExportJobRepository struct {
	mu      sync.RWMutex
	records map[string]domain.ReferralExportJob
}

func (r *ExportJobRepository) Create(_ context.Context, row domain.ReferralExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.ID]; ok {
		return domain.ErrConflict
	}
	r.records[row.ID] = row
	return nil
}
func (r *ExportJobRepository) Update(_ context.Context, row domain.ReferralExportJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[row.ID]; !ok {
		return domain.ErrNotFound
	}
	r.records[row.ID] = row
	return nil
}
func (r *ExportJobRepository) GetByID(_ context.Context, id string) (domain.ReferralExportJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.records[id]
	if !ok {
		return domain.ReferralExportJob{}, domain.ErrNotFound
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

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func periodStart(period string, now time.Time) time.Time {
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7)
	case "90d":
		return now.AddDate(0, 0, -90)
	case "all":
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return now.AddDate(0, 0, -30)
	}
}
func ltvFromRevenue(revenue float64, conversions int) float64 {
	if conversions <= 0 {
		return 0
	}
	return round2((revenue / float64(conversions)) * 1.25)
}
func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }

func seedWarehouse(w *WarehouseRepository, now time.Time) {
	w.referralEvents[uuid.NewString()] = domain.ReferralEvent{ID: uuid.NewString(), EventID: "evt_seed_click", EventType: "affiliate.click.tracked", ReferralToken: "creator-alpha", Platform: "instagram", Country: "US", OccurredAt: now.Add(-24 * time.Hour), CreatedAt: now}
	dates := []string{now.AddDate(0, 0, -2).Format("2006-01-02"), now.AddDate(0, 0, -1).Format("2006-01-02"), now.Format("2006-01-02")}
	for i, d := range dates {
		w.dailyAggregates["a-"+d] = domain.ReferralAggregateDaily{ID: "a-" + d, Date: d, ReferralToken: "creator-alpha", Platform: "instagram", Clicks: 120 + i*10, Signups: 24 + i*2, Conversions: 10 + i, Revenue: 340 + float64(i*35), UpdatedAt: now}
		w.dailyAggregates["b-"+d] = domain.ReferralAggregateDaily{ID: "b-" + d, Date: d, ReferralToken: "creator-beta", Platform: "youtube", Clicks: 90 + i*8, Signups: 14 + i, Conversions: 7 + i, Revenue: 210 + float64(i*20), UpdatedAt: now}
		w.funnelAggregates["f-"+d] = domain.ReferralFunnelAggregate{ID: "f-" + d, Date: d, Clicks: 300 + i*20, Signups: 60 + i*5, FirstPurchases: 26 + i*2, RepeatPurchases: 8 + i, UpdatedAt: now}
		w.geoAggregates["g-us-"+d] = domain.ReferralGeoAggregate{ID: "g-us-" + d, Date: d, Country: "US", Clicks: 180 + i*10, Conversions: 14 + i, Revenue: 420 + float64(i*30), UpdatedAt: now}
		w.geoAggregates["g-ca-"+d] = domain.ReferralGeoAggregate{ID: "g-ca-" + d, Date: d, Country: "CA", Clicks: 80 + i*5, Conversions: 6 + i, Revenue: 160 + float64(i*15), UpdatedAt: now}
	}
	w.cohorts[now.AddDate(0, 0, -30).Format("2006-01-02")] = domain.ReferralCohortRetention{ID: "c1", CohortDate: now.AddDate(0, 0, -30).Format("2006-01-02"), CohortSize: 120, Day7Rate: 0.58, Day30Rate: 0.31, Day90Rate: 0.12, RepeatPurchaseRate: 0.26, UpdatedAt: now}
	w.cohorts[now.AddDate(0, 0, -14).Format("2006-01-02")] = domain.ReferralCohortRetention{ID: "c2", CohortDate: now.AddDate(0, 0, -14).Format("2006-01-02"), CohortSize: 96, Day7Rate: 0.61, Day30Rate: 0.00, Day90Rate: 0.00, RepeatPurchaseRate: 0.22, UpdatedAt: now}
}
