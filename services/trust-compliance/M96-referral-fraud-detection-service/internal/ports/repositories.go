package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/domain"
)

type ReferralEventRepository interface {
	Create(ctx context.Context, row domain.ReferralEvent) error
	GetByEventID(ctx context.Context, eventID string) (domain.ReferralEvent, error)
	Count() int
}

type FraudDecisionRepository interface {
	Create(ctx context.Context, row domain.FraudDecision) error
	GetByEventID(ctx context.Context, eventID string) (domain.FraudDecision, error)
	GetByDecisionID(ctx context.Context, decisionID string) (domain.FraudDecision, error)
	Update(ctx context.Context, row domain.FraudDecision) error
	ListRecent(ctx context.Context, limit int) ([]domain.FraudDecision, error)
}

type RiskPolicyRepository interface {
	ListActive(ctx context.Context) ([]domain.RiskPolicy, error)
	Upsert(ctx context.Context, row domain.RiskPolicy) error
}

type DeviceFingerprintRepository interface {
	UpsertSeen(ctx context.Context, fingerprintHash, ip string, at time.Time) (domain.DeviceFingerprint, error)
}

type ClusterRepository interface {
	UpsertByKey(ctx context.Context, key, reason string, at time.Time) (domain.Cluster, error)
}

type DisputeCaseRepository interface {
	Create(ctx context.Context, row domain.DisputeCase) error
	GetByDecisionID(ctx context.Context, decisionID string) (domain.DisputeCase, error)
	ListByStatus(ctx context.Context, status string, limit int) ([]domain.DisputeCase, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, row domain.AuditLog) error
	ListRecent(ctx context.Context, limit int) ([]domain.AuditLog, error)
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
