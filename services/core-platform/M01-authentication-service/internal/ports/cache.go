package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// LockoutState is the current lockout envelope for a login key.
// It is cache-backed to avoid hot writes on every failed login.
type LockoutState struct {
	FailedCount int
	LockedUntil *time.Time
}

// LockoutStore handles short-lived brute-force protection state.
type LockoutStore interface {
	Get(ctx context.Context, key string) (LockoutState, error)
	RecordFailure(ctx context.Context, key string, now time.Time, threshold int, lockoutWindow time.Duration) (LockoutState, error)
	Clear(ctx context.Context, key string) error
}

// SessionRevocationStore keeps revocation markers with token-aligned TTL.
// This allows immediate logout semantics without token introspection on every call.
type SessionRevocationStore interface {
	MarkRevoked(ctx context.Context, sessionID uuid.UUID, expiresAt time.Time) error
	IsRevoked(ctx context.Context, sessionID uuid.UUID) (bool, error)
}

// MFAChallenge is a temporary second-factor challenge envelope.
// It includes auth context so final token issuance can avoid an extra user lookup.
type MFAChallenge struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Method    string    `json:"method"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// MFAChallengeStore persists short-lived 2FA challenges.
type MFAChallengeStore interface {
	Put(ctx context.Context, token string, challenge MFAChallenge, ttl time.Duration) error
	Get(ctx context.Context, token string) (*MFAChallenge, error)
	Delete(ctx context.Context, token string) error
}

// OIDCAuthState stores PKCE/state data between authorize and callback.
// This preserves anti-replay and anti-CSRF checks across redirects.
type OIDCAuthState struct {
	Provider     string    `json:"provider"`
	RedirectURI  string    `json:"redirect_uri"`
	Nonce        string    `json:"nonce"`
	LoginHint    string    `json:"login_hint"`
	ClientContext string   `json:"client_context,omitempty"`
	CodeVerifier string    `json:"code_verifier"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OIDCStateStore manages temporary OIDC authorization state.
type OIDCStateStore interface {
	Put(ctx context.Context, state string, value OIDCAuthState, ttl time.Duration) error
	Get(ctx context.Context, state string) (*OIDCAuthState, error)
	Delete(ctx context.Context, state string) error
}

// RegistrationCompletion stores short-lived completion state for deferred OIDC onboarding.
type RegistrationCompletion struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RegistrationCompletionStore persists short-lived completion tokens.
type RegistrationCompletionStore interface {
	Put(ctx context.Context, token string, value RegistrationCompletion, ttl time.Duration) error
	Get(ctx context.Context, token string) (*RegistrationCompletion, error)
	Delete(ctx context.Context, token string) error
}

// CloneJSON deep-copies JSON-serializable values.
// It is used to avoid accidental mutation sharing in cached state objects.
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
