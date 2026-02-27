package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
)

type TransactionListQuery struct {
	UserID string
	Limit  int
	Offset int
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction domain.Transaction) error
	Update(ctx context.Context, transaction domain.Transaction) error
	GetByID(ctx context.Context, transactionID string) (domain.Transaction, error)
	GetByProviderTransactionID(ctx context.Context, providerTransactionID string) (domain.Transaction, error)
	List(ctx context.Context, query TransactionListQuery) ([]domain.Transaction, int, error)
}

type RefundRepository interface {
	Create(ctx context.Context, refund domain.Refund) error
	ListByTransaction(ctx context.Context, transactionID string) ([]domain.Refund, error)
}

type BalanceRepository interface {
	GetOrCreate(ctx context.Context, userID string) (domain.UserBalance, error)
	Upsert(ctx context.Context, balance domain.UserBalance) error
}

type WebhookRepository interface {
	IsDuplicate(ctx context.Context, webhookID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, record domain.Webhook, expiresAt time.Time) error
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}

type OutboxRecord struct {
	RecordID   string
	EventClass string
	Envelope   contracts.EventEnvelope
	CreatedAt  time.Time
	SentAt     *time.Time
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
