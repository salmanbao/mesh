package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/domain"
)

func (s *Service) RenderEmbed(ctx context.Context, in RenderEmbedInput) (RenderedEmbed, error) {
	in.EntityType = strings.ToLower(strings.TrimSpace(in.EntityType))
	in.EntityID = strings.TrimSpace(in.EntityID)
	in.Theme = normalizeTheme(in.Theme)
	in.Color = normalizeColor(in.Color)
	in.ButtonText = normalizeButtonText(in.ButtonText)
	in.Language = normalizeLanguage(in.Language)
	referrerDomain := parseReferrerDomain(in.Referrer)
	if !domain.IsValidEntityType(in.EntityType) || in.EntityID == "" {
		return RenderedEmbed{}, domain.ErrInvalidInput
	}
	now := s.nowFn()
	ipMasked := anonymizeIP(in.ClientIP)
	if ipMasked != "" && s.impressions != nil {
		count, err := s.impressions.CountByIPSince(ctx, ipMasked, now.Add(-1*time.Hour))
		if err != nil {
			return RenderedEmbed{}, err
		}
		if count >= s.cfg.PerIPLimitPerHour {
			return RenderedEmbed{}, domain.ErrRateLimited
		}
	}
	if referrerDomain != "" && s.impressions != nil {
		count, err := s.impressions.CountByEntityReferrerSince(ctx, in.EntityType, in.EntityID, referrerDomain, now.Add(-1*time.Hour))
		if err != nil {
			return RenderedEmbed{}, err
		}
		if count >= s.cfg.PerEmbedLimitPerHour {
			return RenderedEmbed{}, domain.ErrRateLimited
		}
	}
	settings, _ := s.GetOrDefaultSettings(ctx, in.EntityType, in.EntityID)
	if !settings.AllowEmbedding {
		return RenderedEmbed{}, domain.ErrEmbeddingDisabled
	}
	themeUsed := chooseTheme(in.Theme, settings.DefaultTheme)
	colorUsed := in.Color
	if colorUsed == "" {
		colorUsed = settings.PrimaryColor
	}
	buttonText := in.ButtonText
	if buttonText == "" {
		buttonText = settings.CustomButtonText
	}
	if buttonText == "" {
		buttonText = defaultButtonText(in.EntityType)
	}
	cacheKey := cacheKeyFor(in.EntityType, in.EntityID, themeUsed, colorUsed, buttonText, in.Language, in.AutoPlay)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey, now); err == nil && cached.HTML != "" {
			if !in.DNT && s.impressions != nil {
				_ = s.impressions.Append(ctx, domain.Impression{ID: "imp_" + uuid.NewString(), EntityType: in.EntityType, EntityID: in.EntityID, ReferrerDomain: referrerDomain, UserAgentBrowser: browserFamily(in.UserAgent), IPAnonymized: ipMasked, DNTEnabled: false, ThemeUsed: themeUsed, CustomColor: colorUsed, OccurredAt: now})
			}
			return RenderedEmbed{HTML: cached.HTML}, nil
		}
	}
	htmlDoc := renderHTML(in.EntityType, in.EntityID, themeUsed, colorUsed, buttonText, in.AutoPlay, in.Language)
	if s.cache != nil {
		_ = s.cache.Put(ctx, domain.EmbedCache{CacheKey: cacheKey, EntityType: in.EntityType, EntityID: in.EntityID, HTML: htmlDoc, CreatedAt: now, ExpiresAt: now.Add(s.cfg.CacheTTL)})
	}
	if !in.DNT && s.impressions != nil {
		_ = s.impressions.Append(ctx, domain.Impression{ID: "imp_" + uuid.NewString(), EntityType: in.EntityType, EntityID: in.EntityID, ReferrerDomain: referrerDomain, UserAgentBrowser: browserFamily(in.UserAgent), IPAnonymized: ipMasked, DNTEnabled: false, ThemeUsed: themeUsed, CustomColor: colorUsed, OccurredAt: now})
	}
	return RenderedEmbed{HTML: htmlDoc}, nil
}

func (s *Service) GetOrDefaultSettings(ctx context.Context, entityType, entityID string) (domain.EmbedSettings, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	entityID = strings.TrimSpace(entityID)
	if !domain.IsValidEntityType(entityType) || entityID == "" {
		return domain.EmbedSettings{}, domain.ErrInvalidInput
	}
	if s.settings != nil {
		if row, err := s.settings.GetByEntity(ctx, entityType, entityID); err == nil {
			return row, nil
		}
	}
	now := s.nowFn()
	return domain.EmbedSettings{ID: "", EntityType: entityType, EntityID: entityID, AllowEmbedding: true, DefaultTheme: "light", PrimaryColor: "#5B21B6", CustomButtonText: defaultButtonText(entityType), AutoPlayVideo: false, ShowCreatorInfo: true, WhitelistedDomains: []string{}, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, actor Actor, in UpdateEmbedSettingsInput) (domain.EmbedSettings, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.EmbedSettings{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.EmbedSettings{}, domain.ErrIdempotencyRequired
	}
	in.EntityType = strings.ToLower(strings.TrimSpace(in.EntityType))
	in.EntityID = strings.TrimSpace(in.EntityID)
	if !domain.IsValidEntityType(in.EntityType) || in.EntityID == "" {
		return domain.EmbedSettings{}, domain.ErrInvalidInput
	}
	if in.DefaultTheme != "" {
		in.DefaultTheme = normalizeTheme(in.DefaultTheme)
	}
	in.PrimaryColor = normalizeColor(in.PrimaryColor)
	in.CustomButtonText = normalizeButtonText(in.CustomButtonText)
	requestHash := hashJSON(map[string]any{"op": "update_settings", "actor": actor.SubjectID, "entity_type": in.EntityType, "entity_id": in.EntityID, "allow": in.AllowEmbedding, "theme": in.DefaultTheme, "color": in.PrimaryColor, "button": in.CustomButtonText, "autoplay": in.AutoPlayVideo, "show_creator_info": in.ShowCreatorInfo, "domains": normalizeDomains(in.WhitelistedDomains)})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.EmbedSettings{}, err
	} else if ok {
		var out domain.EmbedSettings
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.EmbedSettings{}, err
	}
	row, _ := s.GetOrDefaultSettings(ctx, in.EntityType, in.EntityID)
	now := s.nowFn()
	if row.ID == "" {
		row.ID = "embset_" + uuid.NewString()
		row.CreatedAt = now
	}
	if in.AllowEmbedding != nil {
		row.AllowEmbedding = *in.AllowEmbedding
	}
	if in.DefaultTheme != "" {
		row.DefaultTheme = in.DefaultTheme
	}
	if in.PrimaryColor != "" {
		row.PrimaryColor = in.PrimaryColor
	}
	if in.CustomButtonText != "" {
		row.CustomButtonText = in.CustomButtonText
	}
	if in.AutoPlayVideo != nil {
		row.AutoPlayVideo = *in.AutoPlayVideo
	}
	if in.ShowCreatorInfo != nil {
		row.ShowCreatorInfo = *in.ShowCreatorInfo
	}
	if in.WhitelistedDomains != nil {
		row.WhitelistedDomains = normalizeDomains(in.WhitelistedDomains)
	}
	row.UpdatedAt = now
	row.UpdatedBy = actor.SubjectID
	if s.settings != nil {
		if err := s.settings.Upsert(ctx, row); err != nil {
			return domain.EmbedSettings{}, err
		}
	}
	if s.cache != nil {
		_ = s.cache.DeleteByEntity(ctx, row.EntityType, row.EntityID)
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) GetAnalytics(ctx context.Context, actor Actor, q AnalyticsQuery) (AnalyticsResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return AnalyticsResult{}, domain.ErrUnauthorized
	}
	q.EntityType = strings.ToLower(strings.TrimSpace(q.EntityType))
	q.EntityID = strings.TrimSpace(q.EntityID)
	if !domain.IsValidEntityType(q.EntityType) || q.EntityID == "" {
		return AnalyticsResult{}, domain.ErrInvalidInput
	}
	if q.Granularity == "" {
		q.Granularity = "daily"
	}
	imps, err := s.impressions.ListByEntityRange(ctx, q.EntityType, q.EntityID, q.StartDate, q.EndDate)
	if err != nil {
		return AnalyticsResult{}, err
	}
	ints, err := s.interactions.ListByEntityRange(ctx, q.EntityType, q.EntityID, q.StartDate, q.EndDate)
	if err != nil {
		return AnalyticsResult{}, err
	}
	result := AnalyticsResult{TotalImpressions: len(imps), TotalInteractions: len(ints)}
	if result.TotalImpressions > 0 {
		result.ClickThroughRate = round2(float64(result.TotalInteractions) / float64(result.TotalImpressions) * 100)
	}
	actionCounts := map[string]int{}
	refCounts := map[string]*ReferrerMetric{}
	trend := map[string]*TrendPoint{}
	for _, imp := range imps {
		key := trendBucket(imp.OccurredAt, q.Granularity)
		if trend[key] == nil {
			trend[key] = &TrendPoint{Date: key}
		}
		trend[key].Impressions++
		rd := imp.ReferrerDomain
		if rd == "" {
			rd = "unknown"
		}
		if refCounts[rd] == nil {
			refCounts[rd] = &ReferrerMetric{Domain: rd}
		}
		refCounts[rd].Impressions++
	}
	for _, it := range ints {
		actionCounts[it.Action]++
		key := trendBucket(it.OccurredAt, q.Granularity)
		if trend[key] == nil {
			trend[key] = &TrendPoint{Date: key}
		}
		trend[key].Interactions++
		rd := it.ReferrerDomain
		if rd == "" {
			rd = "unknown"
		}
		if refCounts[rd] == nil {
			refCounts[rd] = &ReferrerMetric{Domain: rd}
		}
		refCounts[rd].Interactions++
	}
	for _, t := range trend {
		if t.Impressions > 0 {
			t.CTR = round2(float64(t.Interactions) / float64(t.Impressions) * 100)
		}
		result.Trend = append(result.Trend, *t)
	}
	sort.Slice(result.Trend, func(i, j int) bool { return result.Trend[i].Date > result.Trend[j].Date })
	for action, count := range actionCounts {
		result.TopActions = append(result.TopActions, ActionMetric{Action: action, Count: count})
	}
	sort.Slice(result.TopActions, func(i, j int) bool { return result.TopActions[i].Count > result.TopActions[j].Count })
	for _, v := range refCounts {
		if v.Impressions > 0 {
			v.CTR = round2(float64(v.Interactions) / float64(v.Impressions) * 100)
		}
		result.ByReferrer = append(result.ByReferrer, *v)
	}
	sort.Slice(result.ByReferrer, func(i, j int) bool { return result.ByReferrer[i].Impressions > result.ByReferrer[j].Impressions })
	return result, nil
}

func (s *Service) GenerateEmbedCode(ctx context.Context, entityType, entityID string) (string, error) {
	settings, err := s.GetOrDefaultSettings(ctx, entityType, entityID)
	if err != nil {
		return "", err
	}
	src := fmt.Sprintf("%s/embed/%s/%s?theme=%s&color=%s", strings.TrimRight(s.cfg.EmbedBaseURL, "/"), settings.EntityType, settings.EntityID, url.QueryEscape(settings.DefaultTheme), url.QueryEscape(settings.PrimaryColor))
	return fmt.Sprintf(`<iframe width="100%%" height="600" src="%s" frameborder="0" allowfullscreen loading="lazy"></iframe>`, src), nil
}

func renderHTML(entityType, entityID, theme, color, buttonText string, autoPlay bool, language string) string {
	bg := "#ffffff"
	fg := "#111827"
	border := "#e5e7eb"
	if theme == "dark" {
		bg = "#111827"
		fg = "#f9fafb"
		border = "#374151"
	}
	if color == "" {
		color = "#5B21B6"
	}
	title := strings.Title(entityType) + " " + entityID
	cta := html.EscapeString(buttonText)
	_ = autoPlay
	_ = language
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>Embed - %s</title><style>body{margin:0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:%s;color:%s}.embed-container{background:%s;border:1px solid %s;border-radius:8px;padding:24px;margin:8px}.cta-button{background-color:%s;color:white;padding:12px 24px;border-radius:6px;border:0;cursor:pointer}</style></head><body><div class="embed-container"><h2>%s</h2><p>Embeddable %s content preview.</p><button class="cta-button" onclick="handleCTA('cta_clicked')">%s</button></div><script>(function(){window.parent.postMessage({event_type:'embed_impression',entity_id:%q,entity_type:%q,timestamp:new Date().toISOString()},'*');window.handleCTA=function(action){window.parent.postMessage({event_type:'embed_click',entity_id:%q,entity_type:%q,action:action,timestamp:new Date().toISOString()},'*');};})();</script></body></html>`, html.EscapeString(title), bg, fg, bg, border, color, html.EscapeString(title), html.EscapeString(entityType), cta, entityID, entityType, entityID, entityType)
}

func normalizeTheme(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "", "auto", "light", "dark":
		if v == "" {
			return "auto"
		}
		return v
	}
	return "auto"
}
func chooseTheme(requestTheme, defaultTheme string) string {
	if requestTheme != "" && requestTheme != "auto" {
		return requestTheme
	}
	if defaultTheme == "dark" {
		return "dark"
	}
	return "light"
}
func normalizeColor(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "#") {
		return ""
	}
	if len(v) != 7 {
		return ""
	}
	for _, ch := range v[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", ch) {
			return ""
		}
	}
	return strings.ToUpper(v)
}
func normalizeButtonText(v string) string {
	v = strings.TrimSpace(v)
	if len([]rune(v)) > 20 {
		return string([]rune(v)[:20])
	}
	return v
}
func defaultButtonText(entityType string) string {
	switch entityType {
	case domain.EntityTypeClip:
		return "Watch Clip"
	case domain.EntityTypeApp:
		return "Open App"
	default:
		return "Join Now"
	}
}
func normalizeLanguage(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) < 2 {
		return "en"
	}
	if len(v) > 2 {
		v = v[:2]
	}
	return v
}
func normalizeDomains(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
		d = strings.TrimSuffix(d, "/")
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}
func parseReferrerDomain(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
func anonymizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) >= 2 {
			return strings.ToLower(parts[0] + ":" + parts[1] + ":*")
		}
		return ""
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*"
	}
	return ""
}
func browserFamily(ua string) string {
	l := strings.ToLower(ua)
	switch {
	case strings.Contains(l, "edg/"):
		return "Edge"
	case strings.Contains(l, "chrome/"):
		return "Chrome"
	case strings.Contains(l, "safari/") && !strings.Contains(l, "chrome/"):
		return "Safari"
	case strings.Contains(l, "firefox/"):
		return "Firefox"
	default:
		return "Unknown"
	}
}
func cacheKeyFor(entityType, entityID, theme, color, buttonText, lang string, autoPlay bool) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s:%t", entityType, entityID, theme, color, buttonText, lang, autoPlay)
}
func trendBucket(t time.Time, granularity string) string {
	t = t.UTC()
	if strings.ToLower(granularity) == "hourly" {
		return t.Format("2006-01-02T15:00:00Z")
	}
	return t.Format("2006-01-02")
}
func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
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
