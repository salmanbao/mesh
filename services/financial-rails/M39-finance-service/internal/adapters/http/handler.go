package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
)

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	transaction, err := h.service.CreateTransaction(r.Context(), actor, application.CreateTransactionInput{
		UserID:                strings.TrimSpace(req.UserID),
		CampaignID:            strings.TrimSpace(req.CampaignID),
		ProductID:             strings.TrimSpace(req.ProductID),
		Provider:              domain.PaymentProvider(strings.ToLower(strings.TrimSpace(req.Provider))),
		ProviderTransactionID: strings.TrimSpace(req.ProviderTransactionID),
		Amount:                req.Amount,
		Currency:              strings.TrimSpace(req.Currency),
		TrafficSource:         strings.TrimSpace(req.TrafficSource),
		UserTier:              strings.TrimSpace(req.UserTier),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", transaction)
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	transactionID := chi.URLParam(r, "id")
	transaction, err := h.service.GetTransaction(r.Context(), actor, transactionID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", transaction)
}

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	query := ports.TransactionListQuery{
		UserID: strings.TrimSpace(r.URL.Query().Get("user_id")),
		Limit:  parseIntOrDefault(r.URL.Query().Get("limit"), 20),
		Offset: parseIntOrDefault(r.URL.Query().Get("offset"), 0),
	}
	out, err := h.service.ListTransactions(r.Context(), actor, query)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"items":      out.Items,
		"pagination": out.Pagination,
	})
}

func (h *Handler) getBalance(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	userID := chi.URLParam(r, "userID")
	balance, err := h.service.GetBalance(r.Context(), actor, userID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", balance)
}

func (h *Handler) createRefund(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateRefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	refund, err := h.service.CreateRefund(r.Context(), actor, application.CreateRefundInput{
		TransactionID: strings.TrimSpace(req.TransactionID),
		UserID:        strings.TrimSpace(req.UserID),
		Amount:        req.Amount,
		Reason:        strings.TrimSpace(req.Reason),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", refund)
}

func (h *Handler) providerWebhook(w http.ResponseWriter, r *http.Request) {
	var req contracts.ProviderWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	transaction, err := h.service.HandleProviderWebhook(r.Context(), application.HandleWebhookInput{
		WebhookID:             strings.TrimSpace(req.WebhookID),
		Provider:              strings.TrimSpace(req.Provider),
		EventType:             strings.TrimSpace(req.EventType),
		ProviderEventID:       strings.TrimSpace(req.ProviderEventID),
		ProviderTransactionID: strings.TrimSpace(req.ProviderTransactionID),
		TransactionID:         strings.TrimSpace(req.TransactionID),
		UserID:                strings.TrimSpace(req.UserID),
		Amount:                req.Amount,
		Currency:              strings.TrimSpace(req.Currency),
		Reason:                strings.TrimSpace(req.Reason),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusAccepted, "", contracts.WebhookAccepted{
		WebhookID: req.WebhookID,
		Status:    string(transaction.Status),
		Processed: time.Now().UTC(),
	})
}

func parseIntOrDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
