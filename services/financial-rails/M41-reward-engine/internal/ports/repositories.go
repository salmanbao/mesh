package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/domain"
)

type RewardRepository interface {
	Save(ctx context.Context, reward domain.Reward) error
	GetBySubmissionID(ctx context.Context, submissionID string) (domain.Reward, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Reward, int, error)
}

type RolloverRepository interface {
	GetByUser(ctx context.Context, userID string) (domain.RolloverBalance, error)
	Upsert(ctx context.Context, balance domain.RolloverBalance) error
}

type SubmissionViewSnapshot struct {
	SubmissionID string
	Views        int64
	PolledAt     time.Time
}

type SnapshotRepository interface {
	Upsert(ctx context.Context, snapshot SubmissionViewSnapshot) error
	Get(ctx context.Context, submissionID string) (SubmissionViewSnapshot, error)
}

type AuditRecord struct {
	LogID        string
	SubmissionID string
	UserID       string
	Action       string
	Amount       float64
	CreatedAt    time.Time
	Metadata     map[string]string
}

type AuditLogRepository interface {
	Append(ctx context.Context, record AuditRecord) error
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
