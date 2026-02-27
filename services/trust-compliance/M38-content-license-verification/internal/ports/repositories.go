package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/domain"
)

type MatchRepository interface {
	Create(ctx context.Context, row domain.CopyrightMatch) error
	GetBySubmissionID(ctx context.Context, submissionID string) (domain.CopyrightMatch, error)
}

type HoldRepository interface {
	Create(ctx context.Context, row domain.LicenseHold) error
	GetBySubmissionID(ctx context.Context, submissionID string) (domain.LicenseHold, error)
	GetByID(ctx context.Context, holdID string) (domain.LicenseHold, error)
}

type AppealRepository interface {
	Create(ctx context.Context, row domain.LicenseAppeal) error
}

type DMCATakedownRepository interface {
	Create(ctx context.Context, row domain.DMCATakedown) error
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
