package domain

import (
	"fmt"
	"strings"
	"time"
)

type InvoiceStatus string

type PaymentStatus string

const (
	InvoiceStatusDraft   InvoiceStatus = "draft"
	InvoiceStatusSent    InvoiceStatus = "sent"
	InvoiceStatusViewed  InvoiceStatus = "viewed"
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusOverdue InvoiceStatus = "overdue"
	InvoiceStatusVoid    InvoiceStatus = "void"

	PaymentStatusUnpaid   PaymentStatus = "unpaid"
	PaymentStatusPartial  PaymentStatus = "partial"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

type Address struct {
	Line1      string `json:"line1"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type InvoiceLineItem struct {
	LineItemID   string  `json:"line_item_id"`
	Description  string  `json:"description"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	Amount       float64 `json:"amount"`
	SourceType   string  `json:"source_type"`
	SourceID     string  `json:"source_id"`
	CurrencyCode string  `json:"currency_code"`
}

type TaxBreakdown struct {
	Amount       float64 `json:"amount"`
	Rate         float64 `json:"rate"`
	Jurisdiction string  `json:"jurisdiction"`
}

type Invoice struct {
	InvoiceID       string            `json:"invoice_id"`
	InvoiceNumber   string            `json:"invoice_number"`
	CustomerID      string            `json:"customer_id"`
	CustomerName    string            `json:"customer_name"`
	CustomerEmail   string            `json:"customer_email"`
	BillingAddress  Address           `json:"billing_address"`
	InvoiceType     string            `json:"invoice_type"`
	Currency        string            `json:"currency"`
	LineItems       []InvoiceLineItem `json:"line_items"`
	Subtotal        float64           `json:"subtotal"`
	Tax             TaxBreakdown      `json:"tax"`
	Total           float64           `json:"total"`
	Status          InvoiceStatus     `json:"status"`
	PaymentStatus   PaymentStatus     `json:"payment_status"`
	Notes           string            `json:"notes"`
	DueDate         time.Time         `json:"due_date"`
	InvoiceDate     time.Time         `json:"invoice_date"`
	PaidDate        *time.Time        `json:"paid_date,omitempty"`
	PaymentMethod   string            `json:"payment_method,omitempty"`
	PDFURL          string            `json:"pdf_url,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	IdempotencyHint string            `json:"-"`
}

type InvoiceEmailEvent struct {
	EventID        string    `json:"event_id"`
	InvoiceID      string    `json:"invoice_id"`
	RecipientEmail string    `json:"recipient_email"`
	Status         string    `json:"status"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type VoidHistory struct {
	VoidID       string    `json:"void_id"`
	InvoiceID    string    `json:"invoice_id"`
	VoidedBy     string    `json:"voided_by"`
	Reason       string    `json:"reason"`
	VoidedAt     time.Time `json:"voided_at"`
	NewInvoiceID string    `json:"new_invoice_id,omitempty"`
}

type InvoicePayment struct {
	PaymentID      string    `json:"payment_id"`
	InvoiceID      string    `json:"invoice_id"`
	TransactionRef string    `json:"transaction_ref"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	Method         string    `json:"method"`
	ProcessedAt    time.Time `json:"processed_at"`
}

type PayoutReceipt struct {
	ReceiptID    string    `json:"receipt_id"`
	PayoutID     string    `json:"payout_id"`
	CreatorID    string    `json:"creator_id"`
	GrossAmount  float64   `json:"gross_amount"`
	PlatformFee  float64   `json:"platform_fee"`
	NetPayout    float64   `json:"net_payout"`
	Currency     string    `json:"currency"`
	PayoutDate   time.Time `json:"payout_date"`
	PayoutStatus string    `json:"payout_status"`
	CreatedAt    time.Time `json:"created_at"`
}

func ValidateCreateInvoiceInput(customerID, customerName, customerEmail string, items []InvoiceLineItem, dueDate time.Time) error {
	if strings.TrimSpace(customerID) == "" {
		return fmt.Errorf("%w: customer_id is required", ErrInvalidInput)
	}
	if strings.TrimSpace(customerName) == "" {
		return fmt.Errorf("%w: customer_name is required", ErrInvalidInput)
	}
	if strings.TrimSpace(customerEmail) == "" {
		return fmt.Errorf("%w: customer_email is required", ErrInvalidInput)
	}
	if len(items) == 0 {
		return fmt.Errorf("%w: at least one line item is required", ErrInvalidInput)
	}
	if dueDate.IsZero() {
		return fmt.Errorf("%w: due_date is required", ErrInvalidInput)
	}
	for _, item := range items {
		if strings.TrimSpace(item.Description) == "" {
			return fmt.Errorf("%w: line item description is required", ErrInvalidInput)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: line item quantity must be positive", ErrInvalidInput)
		}
		if item.UnitPrice <= 0 {
			return fmt.Errorf("%w: line item unit_price must be positive", ErrInvalidInput)
		}
	}
	return nil
}

func ComputeTotals(items []InvoiceLineItem, taxRate float64) (subtotal, tax, total float64) {
	for i := range items {
		amount := float64(items[i].Quantity) * items[i].UnitPrice
		items[i].Amount = amount
		subtotal += amount
	}
	tax = subtotal * taxRate
	total = subtotal + tax
	return subtotal, tax, total
}
