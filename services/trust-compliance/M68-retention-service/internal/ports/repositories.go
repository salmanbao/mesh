package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/domain"
)

type RetentionPolicyRepository interface {
	Create(ctx context.Context, row domain.RetentionPolicy) error
	GetByID(ctx context.Context, policyID string) (domain.RetentionPolicy, error)
	List(ctx context.Context) ([]domain.RetentionPolicy, error)
}

type DeletionPreviewRepository interface {
	Create(ctx context.Context, row domain.DeletionPreview) error
	GetByID(ctx context.Context, previewID string) (domain.DeletionPreview, error)
	Update(ctx context.Context, row domain.DeletionPreview) error
}

type LegalHoldRepository interface {
	Create(ctx context.Context, row domain.LegalHold) error
	List(ctx context.Context, status string) ([]domain.LegalHold, error)
}

type RestorationRepository interface {
	Create(ctx context.Context, row domain.RestorationRequest) error
	GetByID(ctx context.Context, restorationID string) (domain.RestorationRequest, error)
	Update(ctx context.Context, row domain.RestorationRequest) error
	List(ctx context.Context) ([]domain.RestorationRequest, error)
}

type ScheduledDeletionRepository interface {
	Create(ctx context.Context, row domain.ScheduledDeletion) error
	GetByPreviewID(ctx context.Context, previewID string) (domain.ScheduledDeletion, bool, error)
	List(ctx context.Context) ([]domain.ScheduledDeletion, error)
}

type AuditRepository interface {
	Append(ctx context.Context, row domain.AuditLog) error
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
