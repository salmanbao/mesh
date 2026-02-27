package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/domain"
)

type SellerRiskProfileRepository interface {
	Upsert(ctx context.Context, row domain.SellerRiskProfile) error
	GetBySellerID(ctx context.Context, sellerID string) (domain.SellerRiskProfile, error)
}

type SellerEscrowRepository interface {
	Upsert(ctx context.Context, row domain.SellerEscrow) error
	GetBySellerID(ctx context.Context, sellerID string) (domain.SellerEscrow, error)
}

type DisputeLogRepository interface {
	Create(ctx context.Context, row domain.DisputeLog) error
	Update(ctx context.Context, row domain.DisputeLog) error
	GetByID(ctx context.Context, disputeID string) (domain.DisputeLog, error)
	GetByTransactionID(ctx context.Context, transactionID string) (domain.DisputeLog, error)
	ListBySeller(ctx context.Context, sellerID string, limit int) ([]domain.DisputeLog, error)
}

type DisputeEvidenceRepository interface {
	Create(ctx context.Context, row domain.DisputeEvidence) error
	ListByDispute(ctx context.Context, disputeID string) ([]domain.DisputeEvidence, error)
}

type FraudPatternFlagRepository interface {
	Create(ctx context.Context, row domain.FraudPatternFlag) error
	ListBySeller(ctx context.Context, sellerID string, limit int) ([]domain.FraudPatternFlag, error)
}

type ReserveTriggerLogRepository interface {
	Create(ctx context.Context, row domain.ReserveTriggerLog) error
	ListBySeller(ctx context.Context, sellerID string, limit int) ([]domain.ReserveTriggerLog, error)
}

type SellerDebtLogRepository interface {
	Create(ctx context.Context, row domain.SellerDebtLog) error
	ListBySeller(ctx context.Context, sellerID string, limit int) ([]domain.SellerDebtLog, error)
}

type SellerSuspensionLogRepository interface {
	Create(ctx context.Context, row domain.SellerSuspensionLog) error
	ListBySeller(ctx context.Context, sellerID string, limit int) ([]domain.SellerSuspensionLog, error)
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
