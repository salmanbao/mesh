package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) renderEmbed(w http.ResponseWriter, r *http.Request) {
	autoPlay, _ := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("auto_play")))
	out, err := h.service.RenderEmbed(r.Context(), application.RenderEmbedInput{
		EntityType:     chi.URLParam(r, "entity_type"),
		EntityID:       chi.URLParam(r, "entity_id"),
		Theme:          r.URL.Query().Get("theme"),
		Color:          r.URL.Query().Get("color"),
		ButtonText:     r.URL.Query().Get("button_text"),
		AutoPlay:       autoPlay,
		Language:       r.URL.Query().Get("language"),
		Ref:            r.URL.Query().Get("ref"),
		Referrer:       r.Header.Get("Referer"),
		AcceptLanguage: r.Header.Get("Accept-Language"),
		UserAgent:      r.UserAgent(),
		DNT:            strings.TrimSpace(r.Header.Get("DNT")) == "1",
		ClientIP:       clientIP(r),
		RequestID:      requestIDFromContext(r.Context()),
	})
	if err != nil {
		code, c := mapDomainError(err)
		switch err {
		case domain.ErrRateLimited:
			writeJSON(w, code, contracts.ErrorResponse{Error: c, Message: "Too many requests from your IP. Please try again later.", RetryAfterSeconds: 900})
		case domain.ErrEmbeddingDisabled:
			writeJSON(w, code, contracts.ErrorResponse{Error: c, Message: "Embedding disabled for this content by creator."})
		default:
			writeJSON(w, code, contracts.ErrorResponse{Error: c, Message: err.Error()})
		}
		return
	}
	setEmbedSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out.HTML))
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")
	settings, err := h.service.GetOrDefaultSettings(r.Context(), entityType, entityID)
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	analytics, err := h.service.GetAnalytics(r.Context(), actorFromContext(r.Context()), application.AnalyticsQuery{EntityType: entityType, EntityID: entityID, Granularity: "daily"})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	embedCode, _ := h.service.GenerateEmbedCode(r.Context(), entityType, entityID)
	writeSuccess(w, http.StatusOK, contracts.EmbedSettingsResponse{
		EntityType: settings.EntityType, EntityID: settings.EntityID,
		AllowEmbedding: settings.AllowEmbedding, DefaultTheme: settings.DefaultTheme, PrimaryColor: settings.PrimaryColor,
		CustomButtonText: settings.CustomButtonText, AutoPlayVideo: settings.AutoPlayVideo, ShowCreatorInfo: settings.ShowCreatorInfo,
		WhitelistedDomains: settings.WhitelistedDomains, EmbedCode: embedCode, UpdatedAt: settings.UpdatedAt.UTC().Format(time.RFC3339),
		Metrics: contracts.SettingsMetrics{TotalImpressions: analytics.TotalImpressions, TotalInteractions: analytics.TotalInteractions, ClickThroughRate: analytics.ClickThroughRate, TopReferrers: toTopReferrers(analytics.ByReferrer)},
	})
}

func (h *Handler) postSettings(w http.ResponseWriter, r *http.Request) {
	var req contracts.UpdateEmbedSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.UpdateSettings(r.Context(), actorFromContext(r.Context()), application.UpdateEmbedSettingsInput{
		EntityType: chi.URLParam(r, "entity_type"), EntityID: chi.URLParam(r, "entity_id"),
		AllowEmbedding: req.AllowEmbedding, DefaultTheme: req.DefaultTheme, PrimaryColor: req.PrimaryColor, CustomButtonText: req.CustomButtonText,
		AutoPlayVideo: req.AutoPlayVideo, ShowCreatorInfo: req.ShowCreatorInfo, WhitelistedDomains: req.WhitelistedDomains,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	embedCode, _ := h.service.GenerateEmbedCode(r.Context(), row.EntityType, row.EntityID)
	writeSuccess(w, http.StatusOK, contracts.EmbedSettingsResponse{EntityType: row.EntityType, EntityID: row.EntityID, AllowEmbedding: row.AllowEmbedding, DefaultTheme: row.DefaultTheme, PrimaryColor: row.PrimaryColor, CustomButtonText: row.CustomButtonText, AutoPlayVideo: row.AutoPlayVideo, ShowCreatorInfo: row.ShowCreatorInfo, WhitelistedDomains: row.WhitelistedDomains, EmbedCode: embedCode, UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339)})
}

func (h *Handler) getAnalytics(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")
	var fromPtr, toPtr *time.Time
	if raw := strings.TrimSpace(r.URL.Query().Get("start_date")); raw != "" {
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			fromPtr = &t
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("end_date")); raw != "" {
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			tt := t.Add(24*time.Hour - time.Nanosecond)
			toPtr = &tt
		}
	}
	out, err := h.service.GetAnalytics(r.Context(), actorFromContext(r.Context()), application.AnalyticsQuery{EntityType: entityType, EntityID: entityID, StartDate: fromPtr, EndDate: toPtr, Granularity: r.URL.Query().Get("granularity"), GroupBy: r.URL.Query().Get("group_by")})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	acts := make([]contracts.ActionMetric, 0, len(out.TopActions))
	for _, a := range out.TopActions {
		acts = append(acts, contracts.ActionMetric{Action: a.Action, Count: a.Count})
	}
	refs := make([]contracts.ReferrerMetric, 0, len(out.ByReferrer))
	for _, rr := range out.ByReferrer {
		refs = append(refs, contracts.ReferrerMetric{ReferrerDomain: rr.Domain, Impressions: rr.Impressions, Interactions: rr.Interactions, CTR: rr.CTR})
	}
	trend := make([]contracts.TrendPoint, 0, len(out.Trend))
	for _, t := range out.Trend {
		trend = append(trend, contracts.TrendPoint{Date: t.Date, Impressions: t.Impressions, Interactions: t.Interactions, CTR: t.CTR})
	}
	writeSuccess(w, http.StatusOK, contracts.EmbedAnalyticsResponse{Summary: contracts.AnalyticsSummary{TotalImpressions: out.TotalImpressions, TotalInteractions: out.TotalInteractions, ClickThroughRate: out.ClickThroughRate, TopActions: acts}, ByReferrer: refs, Trend: trend})
}

func toTopReferrers(in []application.ReferrerMetric) []contracts.ReferrerMetric {
	out := make([]contracts.ReferrerMetric, 0, len(in))
	for _, r := range in {
		out = append(out, contracts.ReferrerMetric{Domain: r.Domain, Impressions: r.Impressions, Interactions: r.Interactions, CTR: r.CTR})
	}
	return out
}

func setEmbedSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Frame-Options", "ALLOW-FROM https://platform.com")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'self' embed.platform.com;")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func clientIP(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
