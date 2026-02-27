package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
)

type HistoryQuery struct {
	UserID string
	Limit  int
	Offset int
}

type PayoutRepository interface {
	Create(ctx context.Context, payout domain.Payout) error
	Update(ctx context.Context, payout domain.Payout) error
	GetByID(ctx context.Context, payoutID string) (domain.Payout, error)
	List(ctx context.Context, query HistoryQuery) ([]domain.Payout, int, error)
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
