package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
)

type EscrowHoldRepository interface {
	Create(ctx context.Context, row domain.EscrowHold) error
	GetByID(ctx context.Context, escrowID string) (domain.EscrowHold, error)
	Update(ctx context.Context, row domain.EscrowHold) error
}

type LedgerEntryRepository interface {
	Append(ctx context.Context, row domain.LedgerEntry) error
	ListByCampaignID(ctx context.Context, campaignID string) ([]domain.LedgerEntry, error)
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

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
