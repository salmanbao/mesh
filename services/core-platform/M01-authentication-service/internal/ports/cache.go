package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LockoutState struct {
	FailedCount int
	LockedUntil *time.Time
}

type LockoutStore interface {
	Get(ctx context.Context, key string) (LockoutState, error)
	RecordFailure(ctx context.Context, key string, now time.Time, threshold int, lockoutWindow time.Duration) (LockoutState, error)
	Clear(ctx context.Context, key string) error
}

type SessionRevocationStore interface {
	MarkRevoked(ctx context.Context, sessionID uuid.UUID, expiresAt time.Time) error
	IsRevoked(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

type MFAChallenge struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Method    string    `json:"method"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

type MFAChallengeStore interface {
	Put(ctx context.Context, token string, challenge MFAChallenge, ttl time.Duration) error
	Get(ctx context.Context, token string) (*MFAChallenge, error)
	Delete(ctx context.Context, token string) error
}

type OIDCAuthState struct {
	Provider     string    `json:"provider"`
	RedirectURI  string    `json:"redirect_uri"`
	Nonce        string    `json:"nonce"`
	LoginHint    string    `json:"login_hint"`
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type OIDCStateStore interface {
	Put(ctx context.Context, state string, value OIDCAuthState, ttl time.Duration) error
	Get(ctx context.Context, state string) (*OIDCAuthState, error)
	Delete(ctx context.Context, state string) error
}

func CloneJSON[T any](in T) (T, error) {
	var out T
	raw, err := json.Marshal(in)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}
