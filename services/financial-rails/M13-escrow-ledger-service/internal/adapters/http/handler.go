package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
)

type Handler struct{ service *application.Service }
func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createHold(w http.ResponseWriter, r *http.Request) {
	var req contracts.CreateHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	hold, err := h.service.CreateHold(r.Context(), actor, application.CreateHoldInput{CampaignID: req.CampaignID, CreatorID: req.CreatorID, Amount: req.Amount})
	if err != nil { code, c := mapDomainError(err); writeError(w, code, c, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "hold created", toHoldResponse(hold))
}
func (h *Handler) releaseHold(w http.ResponseWriter, r *http.Request) {
	var req contracts.ReleaseHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	hold, err := h.service.Release(r.Context(), actor, application.ReleaseInput{EscrowID: req.EscrowID, Amount: req.Amount})
	if err != nil { code, c := mapDomainError(err); writeError(w, code, c, err.Error(), requestIDFromContext(r.Context())); return }
	resp := toHoldResponse(hold); resp.EventDelivery = "pending"
	writeSuccess(w, http.StatusOK, "release processed", resp)
}
func (h *Handler) refundHold(w http.ResponseWriter, r *http.Request) {
	var req contracts.RefundHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid_input", "invalid json body", requestIDFromContext(r.Context())); return }
	actor := actorFromContext(r.Context())
	hold, err := h.service.Refund(r.Context(), actor, application.RefundInput{EscrowID: req.EscrowID, Amount: req.Amount})
	if err != nil { code, c := mapDomainError(err); writeError(w, code, c, err.Error(), requestIDFromContext(r.Context())); return }
	resp := toHoldResponse(hold); resp.EventDelivery = "pending"
	writeSuccess(w, http.StatusOK, "refund processed", resp)
}
func (h *Handler) walletBalance(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	bal, err := h.service.GetWalletBalance(r.Context(), actor, strings.TrimSpace(r.URL.Query().Get("campaign_id")))
	if err != nil { code, c := mapDomainError(err); writeError(w, code, c, err.Error(), requestIDFromContext(r.Context())); return }
	writeSuccess(w, http.StatusOK, "wallet balance", contracts.WalletBalanceResponse{CampaignID: bal.CampaignID, HeldBalance: bal.HeldBalance, ReleasedBalance: bal.ReleasedBalance, RefundedBalance: bal.RefundedBalance, NetEscrowBalance: bal.NetEscrowBalance})
}

func toHoldResponse(hold domain.EscrowHold) contracts.HoldResponse {
	return contracts.HoldResponse{
		EscrowID:        hold.EscrowID,
		CampaignID:      hold.CampaignID,
		CreatorID:       hold.CreatorID,
		Status:          hold.Status,
		OriginalAmount:  hold.OriginalAmount,
		RemainingAmount: hold.RemainingAmount,
		ReleasedAmount:  hold.ReleasedAmount,
		RefundedAmount:  hold.RefundedAmount,
	}
}
