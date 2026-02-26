package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
)

func (s *Service) CreateInvoice(ctx context.Context, actor Actor, input CreateInvoiceInput) (domain.Invoice, error) {
	if actor.Role != "admin" {
		return domain.Invoice{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Invoice{}, domain.ErrIdempotencyRequired
	}
	if err := domain.ValidateCreateInvoiceInput(input.CustomerID, input.CustomerName, input.CustomerEmail, input.LineItems, input.DueDate); err != nil {
		return domain.Invoice{}, err
	}
	if _, err := s.auth.GetUser(ctx, input.CustomerID); err != nil {
		return domain.Invoice{}, fmt.Errorf("resolve customer: %w", err)
	}
	for _, item := range input.LineItems {
		if item.SourceType != "" && item.SourceID != "" {
			if err := s.catalog.GetSource(ctx, item.SourceType, item.SourceID); err != nil {
				return domain.Invoice{}, fmt.Errorf("resolve source %s/%s: %w", item.SourceType, item.SourceID, err)
			}
		}
	}
	if err := s.subscription.ValidateSubscription(ctx, input.CustomerID); err != nil {
		return domain.Invoice{}, fmt.Errorf("validate subscription scope: %w", err)
	}

	requestHash := hashPayload(input)
	if existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, s.nowFn()); err != nil {
		return domain.Invoice{}, err
	} else if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Invoice{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Invoice
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Invoice{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Invoice{}, err
	}

	taxRate, feeErr := s.fees.GetFeeRate(ctx, input.InvoiceType)
	if feeErr != nil {
		taxRate = s.cfg.DefaultTaxRate
	}
	normalizedItems := make([]domain.InvoiceLineItem, 0, len(input.LineItems))
	for _, item := range input.LineItems {
		item.LineItemID = uuid.NewString()
		item.Amount = float64(item.Quantity) * item.UnitPrice
		if item.CurrencyCode == "" {
			item.CurrencyCode = s.cfg.DefaultCurrency
		}
		normalizedItems = append(normalizedItems, item)
	}
	subtotal, taxAmount, total := domain.ComputeTotals(normalizedItems, taxRate)

	day := s.nowFn()
	sequence, err := s.invoices.NextInvoiceSequence(ctx, day)
	if err != nil {
		return domain.Invoice{}, err
	}
	invoiceNumber := fmt.Sprintf("INV-%s-%06d", day.Format("20060102"), sequence)

	invoice := domain.Invoice{
		InvoiceID:      uuid.NewString(),
		InvoiceNumber:  invoiceNumber,
		CustomerID:     input.CustomerID,
		CustomerName:   input.CustomerName,
		CustomerEmail:  input.CustomerEmail,
		BillingAddress: input.BillingAddress,
		InvoiceType:    input.InvoiceType,
		Currency:       s.cfg.DefaultCurrency,
		LineItems:      normalizedItems,
		Subtotal:       subtotal,
		Tax: domain.TaxBreakdown{
			Amount:       taxAmount,
			Rate:         taxRate,
			Jurisdiction: input.BillingAddress.State,
		},
		Total:         total,
		Status:        domain.InvoiceStatusDraft,
		PaymentStatus: domain.PaymentStatusUnpaid,
		Notes:         input.Notes,
		DueDate:       input.DueDate,
		InvoiceDate:   day,
		CreatedAt:     day,
		UpdatedAt:     day,
	}
	if err := s.invoices.Create(ctx, invoice); err != nil {
		return domain.Invoice{}, err
	}
	if err := s.finance.RecordTransaction(ctx, "invoice_created", invoice.InvoiceID, invoice.Total, invoice.Currency); err != nil {
		return domain.Invoice{}, err
	}

	payload, err := json.Marshal(invoice)
	if err != nil {
		return domain.Invoice{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 201, payload, s.nowFn()); err != nil {
		return domain.Invoice{}, err
	}
	return invoice, nil
}

func (s *Service) GetInvoice(ctx context.Context, actor Actor, invoiceID string) (domain.Invoice, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Invoice{}, domain.ErrUnauthorized
	}
	invoice, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return domain.Invoice{}, err
	}
	if actor.Role != "admin" && invoice.CustomerID != actor.SubjectID {
		return domain.Invoice{}, domain.ErrForbidden
	}
	return invoice, nil
}

func (s *Service) ListUserInvoices(ctx context.Context, actor Actor, query ports.InvoiceQuery) (ListInvoicesOutput, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return ListInvoicesOutput{}, domain.ErrUnauthorized
	}
	invoices, total, err := s.invoices.ListByCustomer(ctx, actor.SubjectID, query)
	if err != nil {
		return ListInvoicesOutput{}, err
	}
	return ListInvoicesOutput{
		Invoices: invoices,
		Pagination: contracts.Pagination{
			Limit:  query.Limit,
			Offset: query.Offset,
			Total:  total,
		},
	}, nil
}

func (s *Service) SearchInvoices(ctx context.Context, actor Actor, query ports.InvoiceQuery) (ListInvoicesOutput, error) {
	if actor.Role != "admin" {
		return ListInvoicesOutput{}, domain.ErrForbidden
	}
	invoices, total, err := s.invoices.Search(ctx, query)
	if err != nil {
		return ListInvoicesOutput{}, err
	}
	return ListInvoicesOutput{
		Invoices: invoices,
		Pagination: contracts.Pagination{
			Limit:  query.Limit,
			Offset: query.Offset,
			Total:  total,
		},
	}, nil
}

func (s *Service) SendInvoice(ctx context.Context, actor Actor, input SendInvoiceInput) error {
	invoice, err := s.GetInvoice(ctx, actor, input.InvoiceID)
	if err != nil {
		return err
	}
	now := s.nowFn()
	invoice.Status = domain.InvoiceStatusSent
	invoice.UpdatedAt = now
	if err := s.invoices.Update(ctx, invoice); err != nil {
		return err
	}
	return s.invoices.RecordEmailEvent(ctx, domain.InvoiceEmailEvent{
		EventID:        uuid.NewString(),
		InvoiceID:      invoice.InvoiceID,
		RecipientEmail: input.RecipientEmail,
		Status:         "sent",
		OccurredAt:     now,
	})
}

func (s *Service) DownloadInvoicePDF(ctx context.Context, actor Actor, invoiceID string) ([]byte, string, error) {
	invoice, err := s.GetInvoice(ctx, actor, invoiceID)
	if err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("%s.pdf", invoice.InvoiceNumber)
	content := []byte(fmt.Sprintf("invoice_id=%s\ninvoice_number=%s\namount=%.2f %s\n", invoice.InvoiceID, invoice.InvoiceNumber, invoice.Total, invoice.Currency))
	return content, filename, nil
}

func (s *Service) VoidInvoice(ctx context.Context, actor Actor, input VoidInvoiceInput) (domain.Invoice, error) {
	if actor.Role != "admin" {
		return domain.Invoice{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Invoice{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashPayload(input)
	if existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, s.nowFn()); err != nil {
		return domain.Invoice{}, err
	} else if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Invoice{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Invoice
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Invoice{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Invoice{}, err
	}

	invoice, err := s.invoices.GetByID(ctx, input.InvoiceID)
	if err != nil {
		return domain.Invoice{}, err
	}
	if invoice.PaymentStatus == domain.PaymentStatusPaid {
		return domain.Invoice{}, domain.ErrInvoiceNotVoidable
	}
	now := s.nowFn()
	invoice.Status = domain.InvoiceStatusVoid
	invoice.UpdatedAt = now
	if err := s.invoices.Update(ctx, invoice); err != nil {
		return domain.Invoice{}, err
	}
	if err := s.invoices.RecordVoid(ctx, domain.VoidHistory{
		VoidID:    uuid.NewString(),
		InvoiceID: invoice.InvoiceID,
		VoidedBy:  actor.SubjectID,
		Reason:    input.Reason,
		VoidedAt:  now,
	}); err != nil {
		return domain.Invoice{}, err
	}
	payload, err := json.Marshal(invoice)
	if err != nil {
		return domain.Invoice{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 200, payload, now); err != nil {
		return domain.Invoice{}, err
	}
	return invoice, nil
}

func (s *Service) CreateRefund(ctx context.Context, actor Actor, input RefundInput) error {
	if actor.Role != "admin" {
		return domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ErrIdempotencyRequired
	}
	requestHash := hashPayload(input)
	if existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, s.nowFn()); err != nil {
		return err
	} else if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL)); err != nil {
		return err
	}

	invoice, err := s.invoices.GetByID(ctx, input.InvoiceID)
	if err != nil {
		return err
	}
	now := s.nowFn()
	invoice.PaymentStatus = domain.PaymentStatusRefunded
	invoice.UpdatedAt = now
	if err := s.invoices.Update(ctx, invoice); err != nil {
		return err
	}
	if err := s.invoices.RecordPayment(ctx, domain.InvoicePayment{
		PaymentID:      uuid.NewString(),
		InvoiceID:      invoice.InvoiceID,
		TransactionRef: "refund-" + uuid.NewString(),
		Amount:         input.Amount,
		Currency:       invoice.Currency,
		Status:         "refunded",
		Method:         "stripe",
		ProcessedAt:    now,
	}); err != nil {
		return err
	}
	if err := s.finance.RecordTransaction(ctx, "refund", invoice.InvoiceID, input.Amount, invoice.Currency); err != nil {
		return err
	}
	return s.idempotency.Complete(ctx, actor.IdempotencyKey, 201, []byte(`{"status":"success"}`), now)
}

func (s *Service) RequestBillingExport(ctx context.Context, actor Actor) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	_ = ctx
	return "export-" + uuid.NewString(), nil
}

func (s *Service) RequestBillingDelete(ctx context.Context, actor Actor, reason string) (string, time.Time, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", time.Time{}, domain.ErrUnauthorized
	}
	_ = reason
	return "del-" + uuid.NewString(), s.nowFn().Add(48 * time.Hour), nil
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
