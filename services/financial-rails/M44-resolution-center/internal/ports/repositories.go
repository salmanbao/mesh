package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
)

type DisputeRepository interface {
	Create(ctx context.Context, row domain.Dispute) error
	Update(ctx context.Context, row domain.Dispute) error
	GetByID(ctx context.Context, disputeID string) (domain.Dispute, error)
	GetOpenByTransactionID(ctx context.Context, transactionID string) (domain.Dispute, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]domain.Dispute, error)
}

type MessageRepository interface {
	Create(ctx context.Context, row domain.DisputeMessage) error
	ListByDispute(ctx context.Context, disputeID string, limit int) ([]domain.DisputeMessage, error)
}

type EvidenceRepository interface {
	CreateMany(ctx context.Context, rows []domain.DisputeEvidence) error
	ListByDispute(ctx context.Context, disputeID string) ([]domain.DisputeEvidence, error)
}

type ApprovalRepository interface {
	Create(ctx context.Context, row domain.DisputeApproval) error
	ListByDispute(ctx context.Context, disputeID string) ([]domain.DisputeApproval, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, row domain.DisputeAuditLog) error
	ListByDispute(ctx context.Context, disputeID string, limit int) ([]domain.DisputeAuditLog, error)
}

type StateHistoryRepository interface {
	Create(ctx context.Context, row domain.DisputeStateHistory) error
	ListByDispute(ctx context.Context, disputeID string) ([]domain.DisputeStateHistory, error)
}

type MediationRepository interface {
	Upsert(ctx context.Context, row domain.DisputeMediation) error
}

type AutoResolutionRuleRepository interface {
	ListEnabled(ctx context.Context) ([]domain.AutoResolutionRule, error)
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
