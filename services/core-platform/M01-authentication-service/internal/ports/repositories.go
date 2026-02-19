package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

type CreateUserTxParams struct {
	Email           string
	PasswordHash    string
	RoleName        string
	EmailVerified   bool
	IdempotencyKey  string
	RegisteredAtUTC time.Time
}

type UserRepository interface {
	CreateWithOutboxTx(ctx context.Context, params CreateUserTxParams, outboxEvent OutboxEvent) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, userID uuid.UUID) (domain.User, error)
}

type SessionCreateParams struct {
	UserID         uuid.UUID
	DeviceName     string
	DeviceOS       string
	IPAddress      string
	UserAgent      string
	ExpiresAt      time.Time
	LastActivityAt time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, params SessionCreateParams) (domain.Session, error)
	GetByID(ctx context.Context, sessionID uuid.UUID) (domain.Session, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Session, error)
	TouchActivity(ctx context.Context, sessionID uuid.UUID, touchedAt time.Time) error
	RevokeByID(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type LoginAttemptRepository interface {
	Insert(ctx context.Context, attempt domain.LoginAttempt) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, since *time.Time, status string) ([]domain.LoginAttempt, error)
}

type OutboxEvent struct {
	EventID      uuid.UUID
	EventType    string
	PartitionKey string
	Payload      []byte
	OccurredAt   time.Time
}

type OutboxRecord struct {
	OutboxID     uuid.UUID
	EventType    string
	PartitionKey string
	Payload      []byte
	RetryCount   int
	LastError    *string
	CreatedAt    time.Time
	PublishedAt  *time.Time
	LastErrorAt  *time.Time
	FirstSeenAt  time.Time
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, event OutboxEvent) error
	FetchUnpublished(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkPublished(ctx context.Context, outboxID uuid.UUID, at time.Time) error
	MarkFailed(ctx context.Context, outboxID uuid.UUID, errMsg string, at time.Time) error
}

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

type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}

type RecoveryRepository interface {
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, tokenHash string, usedAt time.Time) (uuid.UUID, error)
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string, verifiedAt time.Time) (uuid.UUID, error)
}

type CredentialRepository interface {
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, updatedAt time.Time) error
	SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool, updatedAt time.Time) error
	HasPassword(ctx context.Context, userID uuid.UUID) (bool, error)
}

type MFARepository interface {
	ListEnabledMethods(ctx context.Context, userID uuid.UUID) ([]string, error)
	SetMethodEnabled(ctx context.Context, userID uuid.UUID, method string, enabled bool, isPrimary bool, updatedAt time.Time) error
	UpsertTOTPSecret(ctx context.Context, userID uuid.UUID, secretEncrypted []byte, updatedAt time.Time) error
	ReplaceBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string, createdAt time.Time) error
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash string, usedAt time.Time) (bool, error)
}

type OIDCRepository interface {
	FindUserByProviderSubject(ctx context.Context, provider, providerUserID string) (uuid.UUID, error)
	UpsertConnection(ctx context.Context, userID uuid.UUID, provider, providerUserID, email string, isPrimary bool, now time.Time) error
	CountConnections(ctx context.Context, userID uuid.UUID) (int, error)
	DeleteConnection(ctx context.Context, userID uuid.UUID, provider string) (bool, error)
}
