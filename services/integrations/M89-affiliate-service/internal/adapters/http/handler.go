package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/application"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/domain"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createReferralLink(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateReferralLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.CreateReferralLink(r.Context(), actorFromContext(r.Context()), application.CreateReferralLinkInput{
		Channel: req.Channel, UTMSource: req.UTMSource, UTMMedium: req.UTMMedium, UTMCampaign: req.UTMCampaign,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, contracts.CreateReferralLinkResponse{
		LinkID: row.LinkID,
		URL:    "/r/" + row.Token,
	})
}

func (h *Handler) trackClick(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.TrackReferralClick(r.Context(), application.TrackClickInput{
		Token:       chi.URLParam(r, "token"),
		ReferrerURL: strings.TrimSpace(r.Header.Get("Referer")),
		ClientIP:    clientIP(r),
		UserAgent:   strings.TrimSpace(r.UserAgent()),
		CookieID:    cookieValue(r, "affiliate_click_id"),
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "affiliate_click_id", Value: out.CookieID, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600})
	http.Redirect(w, r, out.RedirectURL, http.StatusFound)
}

func (h *Handler) getDashboard(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetDashboard(r.Context(), actorFromContext(r.Context()))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	top := make([]contracts.TopLink, 0, len(out.TopLinks))
	for _, row := range out.TopLinks {
		top = append(top, contracts.TopLink{LinkID: row.LinkID, Clicks: row.Clicks, Channel: row.Channel})
	}
	writeSuccess(w, http.StatusOK, contracts.DashboardResponse{
		AffiliateID:       out.AffiliateID,
		TotalReferrals:    out.TotalReferrals,
		TotalClicks:       out.TotalClicks,
		TotalAttributions: out.TotalAttributions,
		ConversionRate:    out.ConversionRate,
		PendingEarnings:   out.PendingEarnings,
		PaidEarnings:      out.PaidEarnings,
		TopLinks:          top,
	})
}

func (h *Handler) listEarnings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.ListEarnings(r.Context(), actorFromContext(r.Context()))
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	items := make([]contracts.EarningResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, contracts.EarningResponse{
			EarningID: row.EarningID,
			OrderID:   row.OrderID,
			Amount:    row.Amount,
			Status:    row.Status,
			CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeSuccess(w, http.StatusOK, contracts.EarningsListResponse{Items: items})
}

func (h *Handler) createExport(w http.ResponseWriter, r *http.Request) {
	var req contracts.ExportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	out, err := h.service.CreateExport(r.Context(), actorFromContext(r.Context()), req.Format)
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusAccepted, contracts.ExportResponse{ExportID: out.ExportID, Status: out.Status})
}

func (h *Handler) suspendAffiliate(w http.ResponseWriter, r *http.Request) {
	var req contracts.SuspendAffiliateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.SuspendAffiliate(r.Context(), actorFromContext(r.Context()), application.SuspendAffiliateInput{
		AffiliateID: chi.URLParam(r, "affiliate_id"),
		Reason:      req.Reason,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, contracts.SuspendAffiliateResponse{
		AffiliateID: row.AffiliateID,
		Status:      row.Status,
		UpdatedAt:   row.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) manualAttribution(w http.ResponseWriter, r *http.Request) {
	var req contracts.ManualAttributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body")
		return
	}
	row, err := h.service.RecordAttribution(r.Context(), actorFromContext(r.Context()), application.RecordAttributionInput{
		AffiliateID:  chi.URLParam(r, "affiliate_id"),
		ClickID:      req.ClickID,
		OrderID:      req.OrderID,
		ConversionID: req.ConversionID,
		Amount:       req.Amount,
		Currency:     req.Currency,
	})
	if err != nil {
		code, c := mapDomainError(err)
		writeError(w, code, c, err.Error())
		return
	}
	writeSuccess(w, http.StatusCreated, contracts.ManualAttributionResponse{
		AttributionID: row.AttributionID,
		AffiliateID:   row.AffiliateID,
		OrderID:       row.OrderID,
		Amount:        row.Amount,
		AttributedAt:  row.AttributedAt.UTC().Format(time.RFC3339),
	})
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

func cookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func _domainUse(_ domain.Affiliate) {}
