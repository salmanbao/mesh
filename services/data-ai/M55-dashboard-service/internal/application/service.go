package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/ports"
)

func (s *Service) GetDashboard(ctx context.Context, actor Actor, input DashboardQueryInput) (domain.Dashboard, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Dashboard{}, domain.ErrUnauthorized
	}
	role := domain.NormalizeRole(actor.Role)
	deviceType := normalizeDeviceType(input.DeviceType)
	dateRange := normalizeDateRange(input.DateRange)
	timezone := strings.TrimSpace(input.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	cacheKey := fmt.Sprintf("dashboard:%s:%s:%s:%s", actor.SubjectID, role, dateRange, deviceType)
	now := s.nowFn()
	if hit, err := s.cache.Get(ctx, cacheKey, now); err == nil && hit != nil {
		return hit.Dashboard, nil
	}

	ctxRead, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	widgets := make(map[string]domain.Widget, 8)
	degraded := make([]string, 0, 4)
	mu := sync.Mutex{}

	type result struct {
		widgetID string
		source   string
		data     map[string]interface{}
		err      error
	}

	jobs := []func(context.Context) result{
		func(ctx context.Context) result {
			profile, err := s.profile.GetProfile(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "profile", source: "M02-Profile-Service", err: err}
			}
			return result{widgetID: "profile", source: "M02-Profile-Service", data: map[string]interface{}{"user_id": profile.UserID, "role": profile.Role}}
		},
		func(ctx context.Context) result {
			billing, err := s.billing.GetBillingSummary(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "billing", source: "M05-Billing-Service", err: err}
			}
			return result{widgetID: "billing", source: "M05-Billing-Service", data: map[string]interface{}{"pending_balance": billing.PendingBalance, "available_balance": billing.AvailableBalance}}
		},
		func(ctx context.Context) result {
			campaigns, err := s.content.ListCampaigns(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "campaigns", source: "M09-Content-Library-Marketplace", err: err}
			}
			rows := make([]map[string]interface{}, 0, len(campaigns))
			for _, c := range campaigns {
				rows = append(rows, map[string]interface{}{"campaign_id": c.CampaignID, "name": c.Name, "submissions": c.Submissions, "avg_views": c.AverageViews})
			}
			return result{widgetID: "campaigns", source: "M09-Content-Library-Marketplace", data: map[string]interface{}{"items": rows}}
		},
		func(ctx context.Context) result {
			metrics, err := s.analytics.GetDashboardMetrics(ctx, actor.SubjectID, string(role), dateRange)
			if err != nil {
				return result{widgetID: "analytics", source: "M54-Analytics-Service", err: err}
			}
			return result{widgetID: "analytics", source: "M54-Analytics-Service", data: map[string]interface{}{"total_earnings": metrics.TotalEarnings, "total_views": metrics.TotalViews, "submissions": metrics.Submissions}}
		},
		func(ctx context.Context) result {
			finance, err := s.finance.GetFinanceSummary(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "payouts", source: "M39-Finance-Service", err: err}
			}
			return result{widgetID: "payouts", source: "M39-Finance-Service", data: map[string]interface{}{"pending_payouts": finance.PendingPayouts, "last_payout": finance.LastPayout}}
		},
		func(ctx context.Context) result {
			reward, err := s.rewards.GetRewardSummary(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "rewards", source: "M41-Reward-Engine", err: err}
			}
			return result{widgetID: "rewards", source: "M41-Reward-Engine", data: map[string]interface{}{"total_rewards": reward.TotalRewards}}
		},
		func(ctx context.Context) result {
			g, err := s.gamification.GetGamification(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "gamification", source: "M47-Gamification-Service", err: err}
			}
			return result{widgetID: "gamification", source: "M47-Gamification-Service", data: map[string]interface{}{"badges": g.Badges}}
		},
		func(ctx context.Context) result {
			p, err := s.products.GetProductSummary(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "products", source: "M60-Product-Service", err: err}
			}
			return result{widgetID: "products", source: "M60-Product-Service", data: map[string]interface{}{"published_apps": p.PublishedApps, "app_revenue": p.AppRevenue}}
		},
		func(ctx context.Context) result {
			e, err := s.escrow.GetEscrowSummary(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "escrow", source: "M13-Escrow-Ledger-Service", err: err}
			}
			return result{widgetID: "escrow", source: "M13-Escrow-Ledger-Service", data: map[string]interface{}{"locked_amount": e.LockedAmount}}
		},
		func(ctx context.Context) result {
			o, err := s.onboarding.GetOnboarding(ctx, actor.SubjectID)
			if err != nil {
				return result{widgetID: "onboarding", source: "M22-Onboarding-Service", err: err}
			}
			return result{widgetID: "onboarding", source: "M22-Onboarding-Service", data: map[string]interface{}{"is_complete": o.IsComplete}}
		},
	}

	wg := sync.WaitGroup{}
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			out := job(ctxRead)
			mu.Lock()
			defer mu.Unlock()
			if out.err != nil {
				degraded = append(degraded, out.widgetID)
				widgets[out.widgetID] = domain.Widget{WidgetID: out.widgetID, Status: domain.WidgetStatusUnavailable, Source: out.source, Data: map[string]interface{}{"error": "temporarily_unavailable"}}
				return
			}
			widgets[out.widgetID] = domain.Widget{WidgetID: out.widgetID, Status: domain.WidgetStatusOK, Source: out.source, Data: out.data}
		}()
	}
	wg.Wait()

	dashboard := domain.Dashboard{
		UserID:          actor.SubjectID,
		Role:            role,
		DateRange:       dateRange,
		Timezone:        timezone,
		GeneratedAt:     now,
		Widgets:         widgets,
		DegradedWidgets: degraded,
	}

	_ = s.cache.Upsert(ctx, ports.CachedDashboard{
		CacheKey:  cacheKey,
		Dashboard: dashboard,
		ExpiresAt: now.Add(s.cfg.DashboardCacheTTL),
		UpdatedAt: now,
	})
	return dashboard, nil
}

func (s *Service) SaveLayout(ctx context.Context, actor Actor, input SaveLayoutInput) (domain.DashboardLayout, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DashboardLayout{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DashboardLayout{}, domain.ErrIdempotencyRequired
	}
	now := s.nowFn()
	payloadHash := hashPayload(input)

	record, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.DashboardLayout{}, err
	}
	if record != nil {
		if record.RequestHash != payloadHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.DashboardLayout{}, domain.ErrIdempotencyConflict
		}
		var out domain.DashboardLayout
		if err := json.Unmarshal(record.ResponseBody, &out); err != nil {
			return domain.DashboardLayout{}, err
		}
		return out, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, payloadHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.DashboardLayout{}, err
	}

	current, err := s.layouts.GetCurrent(ctx, actor.SubjectID, normalizeDeviceType(input.DeviceType))
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.DashboardLayout{}, err
	}
	version := 1
	if current.LayoutVersion > 0 {
		version = current.LayoutVersion + 1
	}
	items := make([]domain.LayoutItem, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, domain.LayoutItem{WidgetID: strings.TrimSpace(item.WidgetID), Position: item.Position, Size: strings.TrimSpace(item.Size), Visible: item.Visible})
	}
	layout := domain.DashboardLayout{
		LayoutID:      uuid.NewString(),
		UserID:        actor.SubjectID,
		DeviceType:    normalizeDeviceType(input.DeviceType),
		LayoutVersion: version,
		Items:         items,
		LastUpdatedAt: now,
	}
	if err := domain.ValidateLayout(layout); err != nil {
		return domain.DashboardLayout{}, err
	}
	if err := s.layouts.Save(ctx, layout); err != nil {
		return domain.DashboardLayout{}, err
	}
	_ = s.cache.InvalidateByUser(ctx, actor.SubjectID)

	encoded, err := json.Marshal(layout)
	if err != nil {
		return domain.DashboardLayout{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 200, encoded, s.nowFn()); err != nil {
		return domain.DashboardLayout{}, err
	}
	return layout, nil
}

func (s *Service) CreateCustomView(ctx context.Context, actor Actor, input CreateCustomViewInput) (domain.CustomView, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CustomView{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CustomView{}, domain.ErrIdempotencyRequired
	}
	now := s.nowFn()
	payloadHash := hashPayload(input)

	record, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.CustomView{}, err
	}
	if record != nil {
		if record.RequestHash != payloadHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.CustomView{}, domain.ErrIdempotencyConflict
		}
		var out domain.CustomView
		if err := json.Unmarshal(record.ResponseBody, &out); err != nil {
			return domain.CustomView{}, err
		}
		return out, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, payloadHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.CustomView{}, err
	}

	view := domain.CustomView{
		ViewID:           uuid.NewString(),
		UserID:           actor.SubjectID,
		ViewName:         strings.TrimSpace(input.ViewName),
		Role:             domain.NormalizeRole(actor.Role),
		WidgetIDs:        sanitizeWidgetIDs(input.WidgetIDs),
		DateRangeDefault: normalizeDateRange(input.DateRangeDefault),
		IsDefault:        input.SetAsDefault,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := domain.ValidateCustomView(view); err != nil {
		return domain.CustomView{}, err
	}
	if err := s.views.Create(ctx, view); err != nil {
		return domain.CustomView{}, err
	}
	if input.SetAsDefault {
		prefs, err := s.preferences.GetByUser(ctx, actor.SubjectID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return domain.CustomView{}, err
		}
		if prefs.PrefID == "" {
			prefs.PrefID = uuid.NewString()
			prefs.UserID = actor.SubjectID
		}
		prefs.DefaultDateRange = view.DateRangeDefault
		prefs.UpdatedAt = now
		if err := s.preferences.Upsert(ctx, prefs); err != nil {
			return domain.CustomView{}, err
		}
	}
	_ = s.cache.InvalidateByUser(ctx, actor.SubjectID)

	encoded, err := json.Marshal(view)
	if err != nil {
		return domain.CustomView{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 201, encoded, s.nowFn()); err != nil {
		return domain.CustomView{}, err
	}
	return view, nil
}

func (s *Service) RecordCacheInvalidation(ctx context.Context, actor Actor, triggerEvent string, widgets []string) (domain.CacheInvalidation, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CacheInvalidation{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CacheInvalidation{}, domain.ErrIdempotencyRequired
	}
	now := s.nowFn()
	payloadHash := hashPayload(struct {
		TriggerEvent string
		Widgets      []string
	}{TriggerEvent: triggerEvent, Widgets: widgets})

	record, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.CacheInvalidation{}, err
	}
	if record != nil {
		if record.RequestHash != payloadHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.CacheInvalidation{}, domain.ErrIdempotencyConflict
		}
		var out domain.CacheInvalidation
		if err := json.Unmarshal(record.ResponseBody, &out); err != nil {
			return domain.CacheInvalidation{}, err
		}
		return out, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, payloadHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.CacheInvalidation{}, err
	}
	row := domain.CacheInvalidation{
		InvalidationID:  uuid.NewString(),
		UserID:          actor.SubjectID,
		TriggerEvent:    strings.TrimSpace(triggerEvent),
		AffectedWidgets: sanitizeWidgetIDs(widgets),
		InvalidatedAt:   now,
	}
	if row.TriggerEvent == "" {
		row.TriggerEvent = "manual"
	}
	if err := s.invalidations.Add(ctx, row); err != nil {
		return domain.CacheInvalidation{}, err
	}
	if err := s.cache.InvalidateByUser(ctx, actor.SubjectID); err != nil {
		return domain.CacheInvalidation{}, err
	}
	encoded, err := json.Marshal(row)
	if err != nil {
		return domain.CacheInvalidation{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 200, encoded, s.nowFn()); err != nil {
		return domain.CacheInvalidation{}, err
	}
	return row, nil
}

func sanitizeWidgetIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeDateRange(raw string) string {
	rangeValue := strings.ToLower(strings.TrimSpace(raw))
	switch rangeValue {
	case "", "7d", "30d", "90d", "ytd", "custom", "month", "week":
		if rangeValue == "" {
			return "30d"
		}
		if rangeValue == "month" {
			return "30d"
		}
		if rangeValue == "week" {
			return "7d"
		}
		return rangeValue
	default:
		return "30d"
	}
}

func normalizeDeviceType(raw string) string {
	device := strings.ToLower(strings.TrimSpace(raw))
	if device == "mobile" {
		return "mobile"
	}
	return "web"
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
