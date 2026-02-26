package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
)

func (h *Handler) createInvoice(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	items := make([]domain.InvoiceLineItem, 0, len(req.LineItems))
	for _, item := range req.LineItems {
		items = append(items, domain.InvoiceLineItem{
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			SourceType:  item.SourceType,
			SourceID:    item.SourceID,
		})
	}
	invoice, err := h.service.CreateInvoice(r.Context(), actor, application.CreateInvoiceInput{
		CustomerID:    req.CustomerID,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		BillingAddress: domain.Address{
			Line1:      req.BillingAddress.Line1,
			City:       req.BillingAddress.City,
			State:      req.BillingAddress.State,
			PostalCode: req.BillingAddress.PostalCode,
			Country:    req.BillingAddress.Country,
		},
		InvoiceType: req.InvoiceType,
		LineItems:   items,
		DueDate:     req.DueDate,
		Notes:       req.Notes,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", invoice)
}

func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	invoiceID := chi.URLParam(r, "invoice_id")
	invoice, err := h.service.GetInvoice(r.Context(), actor, invoiceID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", invoice)
}

func (h *Handler) listUserInvoices(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	query := parseInvoiceQuery(r)
	out, err := h.service.ListUserInvoices(r.Context(), actor, query)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"invoices":   out.Invoices,
		"pagination": out.Pagination,
	})
}

func (h *Handler) sendInvoice(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	invoiceID := chi.URLParam(r, "invoice_id")
	var req contracts.SendInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	err := h.service.SendInvoice(r.Context(), actor, application.SendInvoiceInput{
		InvoiceID:      invoiceID,
		RecipientEmail: req.RecipientEmail,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "Invoice email sent successfully", nil)
}

func (h *Handler) downloadInvoice(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	invoiceID := chi.URLParam(r, "invoice_id")
	content, filename, err := h.service.DownloadInvoicePDF(r.Context(), actor, invoiceID)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) voidInvoice(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	invoiceID := chi.URLParam(r, "invoice_id")
	var req contracts.VoidInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	invoice, err := h.service.VoidInvoice(r.Context(), actor, application.VoidInvoiceInput{
		InvoiceID: invoiceID,
		Reason:    req.Reason,
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"invoice_id": invoice.InvoiceID,
		"status":     invoice.Status,
		"voided_at":  invoice.UpdatedAt,
	})
}

func (h *Handler) searchInvoices(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	query := parseInvoiceQuery(r)
	out, err := h.service.SearchInvoices(r.Context(), actor, query)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", map[string]interface{}{
		"invoices":   out.Invoices,
		"pagination": out.Pagination,
	})
}

func (h *Handler) requestBillingExport(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	exportID, err := h.service.RequestBillingExport(r.Context(), actor)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "Export will be emailed within 48 hours", map[string]string{"export_id": exportID})
}

func (h *Handler) requestBillingDelete(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	deletionID, deadline, err := h.service.RequestBillingDelete(r.Context(), actor, req.Reason)
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusAccepted, "Deletion request submitted. Confirmation email sent.", map[string]interface{}{
		"deletion_id":           deletionID,
		"confirmation_deadline": deadline,
	})
}

func (h *Handler) createRefund(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	if err := h.service.CreateRefund(r.Context(), actor, application.RefundInput{
		InvoiceID:  req.InvoiceID,
		LineItemID: req.LineItemID,
		Amount:     req.Amount,
		Reason:     req.Reason,
	}); err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "Refund processed", nil)
}

func parseInvoiceQuery(r *http.Request) ports.InvoiceQuery {
	query := ports.InvoiceQuery{
		Status:        r.URL.Query().Get("status"),
		InvoiceNumber: r.URL.Query().Get("invoice_number"),
		CustomerEmail: r.URL.Query().Get("customer_email"),
		Limit:         parseIntOrDefault(r.URL.Query().Get("limit"), 20),
		Offset:        parseIntOrDefault(r.URL.Query().Get("offset"), 0),
		MinAmount:     parseFloatOrDefault(r.URL.Query().Get("min_amount"), 0),
		MaxAmount:     parseFloatOrDefault(r.URL.Query().Get("max_amount"), 0),
	}
	if raw := r.URL.Query().Get("date_from"); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			query.DateFrom = &parsed
		}
	}
	if raw := r.URL.Query().Get("date_to"); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			query.DateTo = &parsed
		}
	}
	return query
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

func parseFloatOrDefault(raw string, fallback float64) float64 {
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}
