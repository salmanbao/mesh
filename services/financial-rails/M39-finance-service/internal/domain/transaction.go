package domain

import (
	"strings"
	"time"
)

type TransactionStatus string
type PaymentProvider string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusSucceeded TransactionStatus = "succeeded"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusRefunded  TransactionStatus = "refunded"
)

const (
	ProviderStripe  PaymentProvider = "stripe"
	ProviderPayPal  PaymentProvider = "paypal"
	ProviderMoonPay PaymentProvider = "moonpay"
)

type Transaction struct {
	TransactionID         string            `json:"transaction_id"`
	UserID                string            `json:"user_id"`
	CampaignID            string            `json:"campaign_id"`
	ProductID             string            `json:"product_id"`
	Provider              PaymentProvider   `json:"provider"`
	ProviderTransactionID string            `json:"provider_transaction_id"`
	Amount                float64           `json:"amount"`
	Currency              string            `json:"currency"`
	PlatformFeeRate       float64           `json:"platform_fee_rate"`
	Status                TransactionStatus `json:"status"`
	FailureReason         string            `json:"failure_reason,omitempty"`
	IdempotencyKey        string            `json:"idempotency_key"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	SucceededAt           *time.Time        `json:"succeeded_at,omitempty"`
	FailedAt              *time.Time        `json:"failed_at,omitempty"`
	RefundedAt            *time.Time        `json:"refunded_at,omitempty"`
}

type Refund struct {
	RefundID       string    `json:"refund_id"`
	TransactionID  string    `json:"transaction_id"`
	UserID         string    `json:"user_id"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Reason         string    `json:"reason"`
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
}

type UserBalance struct {
	BalanceID         string    `json:"balance_id"`
	UserID            string    `json:"user_id"`
	AvailableBalance  float64   `json:"available_balance"`
	PendingBalance    float64   `json:"pending_balance"`
	ReservedBalance   float64   `json:"reserved_balance"`
	NegativeBalance   float64   `json:"negative_balance"`
	Currency          string    `json:"currency"`
	LastTransactionID string    `json:"last_transaction_id,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Webhook struct {
	WebhookID             string    `json:"webhook_id"`
	Provider              string    `json:"provider"`
	EventType             string    `json:"event_type"`
	ProviderEventID       string    `json:"provider_event_id"`
	ProviderTransactionID string    `json:"provider_transaction_id"`
	TransactionID         string    `json:"transaction_id"`
	Status                string    `json:"status"`
	ReceivedAt            time.Time `json:"received_at"`
	ProcessedAt           time.Time `json:"processed_at"`
}

func ValidateCreateTransactionInput(userID, campaignID, productID string, provider PaymentProvider, amount float64, currency string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(campaignID) == "" || strings.TrimSpace(productID) == "" {
		return ErrInvalidInput
	}
	if amount <= 0 {
		return ErrInvalidInput
	}
	switch provider {
	case ProviderStripe, ProviderPayPal, ProviderMoonPay:
	default:
		return ErrInvalidInput
	}
	if len(strings.TrimSpace(currency)) != 3 {
		return ErrInvalidInput
	}
	return nil
}

func ValidateRefundInput(transactionID, userID, reason string, amount float64) error {
	if strings.TrimSpace(transactionID) == "" || strings.TrimSpace(userID) == "" {
		return ErrInvalidInput
	}
	if amount <= 0 {
		return ErrInvalidInput
	}
	if strings.TrimSpace(reason) == "" {
		return ErrInvalidInput
	}
	return nil
}

func ValidateWebhookInput(webhookID, provider, eventType string) error {
	if strings.TrimSpace(webhookID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(eventType) == "" {
		return ErrInvalidInput
	}
	return nil
}

func IsSuccessWebhook(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "payment_intent.succeeded", "charge.succeeded", "payment.sale.completed", "transaction.success":
		return true
	default:
		return false
	}
}

func IsFailureWebhook(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "payment_intent.payment_failed", "charge.failed", "payment.sale.denied", "transaction.failed":
		return true
	default:
		return false
	}
}

func IsRefundWebhook(eventType string) bool {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "charge.refunded", "payment.sale.refunded", "refund.completed":
		return true
	default:
		return false
	}
}
