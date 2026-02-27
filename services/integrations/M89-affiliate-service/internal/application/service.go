package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/domain"
)

func (s *Service) CreateReferralLink(ctx context.Context, actor Actor, in CreateReferralLinkInput) (domain.ReferralLink, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ReferralLink{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ReferralLink{}, domain.ErrIdempotencyRequired
	}
	in.Channel = strings.ToLower(strings.TrimSpace(in.Channel))
	in.UTMSource = strings.TrimSpace(in.UTMSource)
	in.UTMMedium = strings.TrimSpace(in.UTMMedium)
	in.UTMCampaign = strings.TrimSpace(in.UTMCampaign)
	requestHash := hashJSON(map[string]any{"op": "create_link", "user": actor.SubjectID, "channel": in.Channel, "utm_source": in.UTMSource, "utm_medium": in.UTMMedium, "utm_campaign": in.UTMCampaign})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ReferralLink{}, err
	} else if ok {
		var out domain.ReferralLink
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ReferralLink{}, err
	}
	aff, err := s.ensureAffiliate(ctx, actor.SubjectID)
	if err != nil {
		return domain.ReferralLink{}, err
	}
	now := s.nowFn()
	link := domain.ReferralLink{LinkID: "link_" + uuid.NewString(), AffiliateID: aff.AffiliateID, Token: randomToken(), Channel: in.Channel, UTMSource: in.UTMSource, UTMMedium: in.UTMMedium, UTMCampaign: in.UTMCampaign, DestinationURL: s.cfg.PublicBaseURL, CreatedAt: now}
	if err := s.links.Create(ctx, link); err != nil {
		return domain.ReferralLink{}, err
	}
	_ = s.appendAudit(ctx, aff.AffiliateID, "affiliate.link.created", actor.SubjectID, "", map[string]string{"link_id": link.LinkID, "channel": link.Channel})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, link)
	return link, nil
}

func (s *Service) TrackReferralClick(ctx context.Context, in TrackClickInput) (TrackClickResult, error) {
	in.Token = strings.TrimSpace(in.Token)
	if in.Token == "" {
		return TrackClickResult{}, domain.ErrInvalidInput
	}
	link, err := s.links.GetByToken(ctx, in.Token)
	if err != nil {
		return TrackClickResult{}, err
	}
	if in.CookieID == "" {
		in.CookieID = uuid.NewString()
	}
	now := s.nowFn()
	click := domain.ReferralClick{ClickID: "click_" + uuid.NewString(), LinkID: link.LinkID, AffiliateID: link.AffiliateID, ReferrerURL: strings.TrimSpace(in.ReferrerURL), IPHash: sha256Hex(strings.TrimSpace(in.ClientIP)), UserAgentHash: sha256Hex(strings.TrimSpace(in.UserAgent)), CookieID: in.CookieID, ClickedAt: now}
	if err := s.clicks.Append(ctx, click); err != nil {
		return TrackClickResult{}, err
	}
	_ = s.enqueueAffiliateClickTracked(ctx, click, uuid.NewString(), now)
	return TrackClickResult{RedirectURL: link.DestinationURL, CookieID: click.CookieID, AffiliateID: link.AffiliateID, LinkID: link.LinkID}, nil
}

func (s *Service) GetDashboard(ctx context.Context, actor Actor) (Dashboard, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return Dashboard{}, domain.ErrUnauthorized
	}
	aff, err := s.ensureAffiliate(ctx, actor.SubjectID)
	if err != nil {
		return Dashboard{}, err
	}
	clicks, err := s.clicks.ListByAffiliateID(ctx, aff.AffiliateID)
	if err != nil {
		return Dashboard{}, err
	}
	attrs, err := s.attributions.ListByAffiliateID(ctx, aff.AffiliateID)
	if err != nil {
		return Dashboard{}, err
	}
	pending, _ := s.earnings.SumByAffiliateAndStatus(ctx, aff.AffiliateID, "pending")
	paid, _ := s.earnings.SumByAffiliateAndStatus(ctx, aff.AffiliateID, "paid")
	links, _ := s.links.ListByAffiliateID(ctx, aff.AffiliateID)
	countByLink := map[string]int{}
	for _, c := range clicks {
		countByLink[c.LinkID]++
	}
	top := make([]TopLinkMetric, 0, len(links))
	for _, l := range links {
		top = append(top, TopLinkMetric{LinkID: l.LinkID, Clicks: countByLink[l.LinkID], Channel: l.Channel})
	}
	if len(top) > 1 {
		for i := 0; i < len(top)-1; i++ {
			for j := i + 1; j < len(top); j++ {
				if top[j].Clicks > top[i].Clicks {
					top[i], top[j] = top[j], top[i]
				}
			}
		}
	}
	conv := 0.0
	if len(clicks) > 0 {
		conv = round2(float64(len(attrs)) / float64(len(clicks)) * 100)
	}
	return Dashboard{AffiliateID: aff.AffiliateID, TotalReferrals: len(links), TotalClicks: len(clicks), TotalAttributions: len(attrs), ConversionRate: conv, PendingEarnings: pending, PaidEarnings: paid, TopLinks: top}, nil
}

func (s *Service) ListEarnings(ctx context.Context, actor Actor) ([]domain.AffiliateEarning, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	aff, err := s.ensureAffiliate(ctx, actor.SubjectID)
	if err != nil {
		return nil, err
	}
	return s.earnings.ListByAffiliateID(ctx, aff.AffiliateID)
}

func (s *Service) CreateExport(ctx context.Context, actor Actor, format string) (ExportResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return ExportResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return ExportResult{}, domain.ErrIdempotencyRequired
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		return ExportResult{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "create_export", "user": actor.SubjectID, "format": format})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return ExportResult{}, err
	} else if ok {
		var out ExportResult
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return ExportResult{}, err
	}
	aff, err := s.ensureAffiliate(ctx, actor.SubjectID)
	if err != nil {
		return ExportResult{}, err
	}
	res := ExportResult{ExportID: "exp_" + uuid.NewString(), Status: "queued"}
	_ = s.appendAudit(ctx, aff.AffiliateID, "affiliate.export.requested", actor.SubjectID, "", map[string]string{"export_id": res.ExportID, "format": format})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 202, res)
	return res, nil
}

func (s *Service) SuspendAffiliate(ctx context.Context, actor Actor, in SuspendAffiliateInput) (domain.Affiliate, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Affiliate{}, domain.ErrUnauthorized
	}
	if !isAdmin(actor) {
		return domain.Affiliate{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Affiliate{}, domain.ErrIdempotencyRequired
	}
	in.AffiliateID = strings.TrimSpace(in.AffiliateID)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.AffiliateID == "" {
		return domain.Affiliate{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "suspend_affiliate", "actor": actor.SubjectID, "affiliate_id": in.AffiliateID, "reason": in.Reason})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Affiliate{}, err
	} else if ok {
		var out domain.Affiliate
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Affiliate{}, err
	}
	aff, err := s.affiliates.GetByID(ctx, in.AffiliateID)
	if err != nil {
		return domain.Affiliate{}, err
	}
	aff.Status = "suspended"
	aff.UpdatedAt = s.nowFn()
	if err := s.affiliates.Update(ctx, aff); err != nil {
		return domain.Affiliate{}, err
	}
	_ = s.appendAudit(ctx, aff.AffiliateID, "affiliate.suspended", actor.SubjectID, in.Reason, nil)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, aff)
	return aff, nil
}

func (s *Service) RecordAttribution(ctx context.Context, actor Actor, in RecordAttributionInput) (domain.ReferralAttribution, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ReferralAttribution{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ReferralAttribution{}, domain.ErrIdempotencyRequired
	}
	in.AffiliateID = strings.TrimSpace(in.AffiliateID)
	in.ClickID = strings.TrimSpace(in.ClickID)
	in.OrderID = strings.TrimSpace(in.OrderID)
	in.ConversionID = strings.TrimSpace(in.ConversionID)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Currency == "" {
		in.Currency = "USD"
	}
	if in.AffiliateID == "" || in.OrderID == "" || in.ConversionID == "" || in.Amount <= 0 {
		return domain.ReferralAttribution{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "record_attribution", "affiliate_id": in.AffiliateID, "order_id": in.OrderID, "conversion_id": in.ConversionID, "amount": in.Amount, "currency": in.Currency})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ReferralAttribution{}, err
	} else if ok {
		var out domain.ReferralAttribution
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ReferralAttribution{}, err
	}
	if _, err := s.attributions.GetByOrderID(ctx, in.OrderID); err == nil {
		return domain.ReferralAttribution{}, domain.ErrConflict
	}
	aff, err := s.affiliates.GetByID(ctx, in.AffiliateID)
	if err != nil {
		return domain.ReferralAttribution{}, err
	}
	now := s.nowFn()
	attr := domain.ReferralAttribution{AttributionID: "attr_" + uuid.NewString(), AffiliateID: in.AffiliateID, ClickID: in.ClickID, ConversionID: in.ConversionID, OrderID: in.OrderID, Amount: in.Amount, Currency: in.Currency, AttributedAt: now}
	if err := s.attributions.Create(ctx, attr); err != nil {
		return domain.ReferralAttribution{}, err
	}
	earningAmount := round2(in.Amount * aff.DefaultRate)
	earning := domain.AffiliateEarning{EarningID: "earn_" + uuid.NewString(), AffiliateID: aff.AffiliateID, AttributionID: attr.AttributionID, OrderID: attr.OrderID, Amount: earningAmount, Status: "pending", CreatedAt: now, UpdatedAt: now}
	if err := s.earnings.Create(ctx, earning); err != nil {
		return domain.ReferralAttribution{}, err
	}
	aff.BalancePending = round2(aff.BalancePending + earningAmount)
	aff.UpdatedAt = now
	if err := s.affiliates.Update(ctx, aff); err != nil {
		return domain.ReferralAttribution{}, err
	}
	if aff.BalancePending >= s.cfg.PayoutThreshold && s.payouts != nil {
		_ = s.payouts.Create(ctx, domain.AffiliatePayout{PayoutID: "pay_" + uuid.NewString(), AffiliateID: aff.AffiliateID, Amount: aff.BalancePending, Status: "queued", QueuedAt: now, UpdatedAt: now})
	}
	_ = s.appendAudit(ctx, aff.AffiliateID, "affiliate.attribution.recorded", actor.SubjectID, "", map[string]string{"order_id": attr.OrderID, "conversion_id": attr.ConversionID})
	_ = s.enqueueAffiliateAttributionCreated(ctx, attr, uuid.NewString(), now)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, attr)
	return attr, nil
}

func (s *Service) ensureAffiliate(ctx context.Context, userID string) (domain.Affiliate, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return domain.Affiliate{}, domain.ErrInvalidInput
	}
	if row, err := s.affiliates.GetByUserID(ctx, userID); err == nil {
		return row, nil
	}
	now := s.nowFn()
	row := domain.Affiliate{AffiliateID: "aff_" + uuid.NewString(), UserID: userID, Status: "active", DefaultRate: s.cfg.CommissionRate, BalancePending: 0, BalancePaid: 0, CreatedAt: now, UpdatedAt: now}
	if err := s.affiliates.Create(ctx, row); err != nil {
		if ex, err2 := s.affiliates.GetByUserID(ctx, userID); err2 == nil {
			return ex, nil
		}
		return domain.Affiliate{}, err
	}
	return row, nil
}

func (s *Service) appendAudit(ctx context.Context, affiliateID, action, actorID, reason string, meta map[string]string) error {
	if s.auditLogs == nil {
		return nil
	}
	return s.auditLogs.Append(ctx, domain.AffiliateAuditLog{AuditLogID: "audit_" + uuid.NewString(), AffiliateID: affiliateID, Action: action, ActorID: actorID, Reason: reason, Metadata: meta, CreatedAt: s.nowFn()})
}

func isAdmin(actor Actor) bool  { return strings.ToLower(strings.TrimSpace(actor.Role)) == "admin" }
func round2(v float64) float64  { return float64(int(v*100+0.5)) / 100 }
func sha256Hex(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func randomToken() string { return strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "") }
func (s *Service) getIdempotent(ctx context.Context, key, expectedHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != expectedHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return rec.ResponseBody, true, nil
}
func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}
func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, v any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(v)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}
