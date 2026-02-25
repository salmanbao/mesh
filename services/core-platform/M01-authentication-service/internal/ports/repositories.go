package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

// CreateUserTxParams captures atomic user-creation inputs.
// It includes outbox/idempotency metadata so registration can be durable and replay-safe.
type CreateUserTxParams struct {
	Email           string
	PasswordHash    string
	RoleName        string
	EmailVerified   bool
	IdempotencyKey  string
	RegisteredAtUTC time.Time
}

// UserRepository defines persistence operations for user identities.
// The transactional create method exists to enforce user+outbox consistency.
type UserRepository interface {
	CreateWithOutboxTx(ctx context.Context, params CreateUserTxParams, outboxEvent OutboxEvent) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, userID uuid.UUID) (domain.User, error)
	Deactivate(ctx context.Context, userID uuid.UUID, deactivatedAt time.Time) error
}

// SessionCreateParams captures metadata required to create a session record.
// Device and network fields are stored for auditability and risk analysis.
type SessionCreateParams struct {
	UserID         uuid.UUID
	DeviceName     string
	DeviceOS       string
	IPAddress      string
	UserAgent      string
	ExpiresAt      time.Time
	LastActivityAt time.Time
}

// SessionRepository manages persistent session lifecycle.
// It is separate from token parsing so revocation and activity tracking remain source-of-truth driven.
type SessionRepository interface {
	Create(ctx context.Context, params SessionCreateParams) (domain.Session, error)
	GetByID(ctx context.Context, sessionID uuid.UUID) (domain.Session, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Session, error)
	TouchActivity(ctx context.Context, sessionID uuid.UUID, touchedAt time.Time) error
	RevokeByID(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

// LoginAttemptRepository stores login outcomes used by lockout and history endpoints.
type LoginAttemptRepository interface {
	Insert(ctx context.Context, attempt domain.LoginAttempt) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, since *time.Time, status string) ([]domain.LoginAttempt, error)
}

// OutboxEvent is the write-side event payload prior to storage.
// It is adapter-neutral to keep application code independent of broker specifics.
type OutboxEvent struct {
	EventID      uuid.UUID
	EventType    string
	PartitionKey string
	Payload      []byte
	OccurredAt   time.Time
}

// OutboxRecord represents durable outbox state, including retry/error metadata.
type OutboxRecord struct {
	OutboxID       uuid.UUID
	EventType      string
	PartitionKey   string
	Payload        []byte
	RetryCount     int
	LastError      *string
	CreatedAt      time.Time
	PublishedAt    *time.Time
	LastErrorAt    *time.Time
	FirstSeenAt    time.Time
	ClaimToken     *string
	ClaimUntil     *time.Time
	DeadLetteredAt *time.Time
}

// OutboxRepository controls publish-retry workflow for domain events.
// This explicit contract enables transactional outbox patterns without leaking DB details.
type OutboxRepository interface {
	Enqueue(ctx context.Context, event OutboxEvent) error
	ClaimUnpublished(ctx context.Context, limit int, claimToken string, claimUntil time.Time) ([]OutboxRecord, error)
	MarkPublished(ctx context.Context, outboxID uuid.UUID, claimToken string, at time.Time) error
	MarkFailed(ctx context.Context, outboxID uuid.UUID, claimToken, errMsg string, at time.Time) error
	MarkDeadLettered(ctx context.Context, outboxID uuid.UUID, claimToken, errMsg string, at time.Time) error
}

// IdempotencyRecord tracks a previously accepted mutating request.
// Storing response metadata lets handlers return stable replay responses.
type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	Status       string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IdempotencyRepository enforces idempotent mutation semantics.
type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}

// RecoveryRepository owns password/email token lifecycle.
// Separate methods for create/consume keep one-time-token invariants explicit.
type RecoveryRepository interface {
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, tokenHash string, usedAt time.Time) (uuid.UUID, error)
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string, verifiedAt time.Time) (uuid.UUID, error)
}

// CredentialRepository manages mutable credential state.
type CredentialRepository interface {
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, updatedAt time.Time) error
	SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool, updatedAt time.Time) error
	HasPassword(ctx context.Context, userID uuid.UUID) (bool, error)
}

// MFARepository controls second-factor enrollment and verification artifacts.
// It centralizes method state to prevent split-brain 2FA behavior across adapters.
type MFARepository interface {
	ListEnabledMethods(ctx context.Context, userID uuid.UUID) ([]string, error)
	SetMethodEnabled(ctx context.Context, userID uuid.UUID, method string, enabled bool, isPrimary bool, updatedAt time.Time) error
	UpsertTOTPSecret(ctx context.Context, userID uuid.UUID, secretEncrypted []byte, updatedAt time.Time) error
	ReplaceBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string, createdAt time.Time) error
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash string, usedAt time.Time) (bool, error)
}

// OIDCRepository persists OIDC account links.
// This keeps provider linkage under M01 ownership rather than in external identity systems.
type OIDCRepository interface {
	FindUserByIssuerSubject(ctx context.Context, issuer, subject string) (uuid.UUID, error)
	UpsertConnection(ctx context.Context, userID uuid.UUID, issuer, subject, provider, email string, isPrimary bool, now time.Time) error
	CountConnections(ctx context.Context, userID uuid.UUID) (int, error)
	DeleteConnection(ctx context.Context, userID uuid.UUID, provider string) (bool, error)
	UpsertToken(ctx context.Context, userID uuid.UUID, provider, accessToken, refreshToken string, expiresAt *time.Time, now time.Time) error
	ListTokensExpiringBefore(ctx context.Context, before time.Time, limit int) ([]OIDCTokenRecord, error)
	UpdateConnectionStatus(ctx context.Context, userID uuid.UUID, provider, status string, now time.Time) error
}

// OIDCTokenRecord is the refreshable OIDC token state owned by M01.
type OIDCTokenRecord struct {
	UserID       uuid.UUID
	Provider     string
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}
