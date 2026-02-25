package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PasswordHasher abstracts password hashing so application code is algorithm-agnostic.
// This boundary exists to allow stronger algorithms/cost tuning without business-layer changes.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// AuthClaims is the normalized token payload used inside M01.
// Keeping this explicit prevents adapters from depending on JWT-library-specific claim shapes.
type AuthClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	SessionID uuid.UUID `json:"session_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	KeyID     string    `json:"kid"`
}

// TokenSigner handles token issuance and validation.
// The application depends on this port to keep signing/parsing concerns in adapters.
type TokenSigner interface {
	Sign(claims AuthClaims) (string, error)
	ParseAndValidate(token string) (AuthClaims, error)
	PublicJWKs() ([]map[string]any, error)
}

// OIDCVerifier encapsulates provider-specific OIDC flows.
// This keeps protocol details at the edge and preserves testability in application services.
type OIDCVerifier interface {
	BuildAuthorizeURL(
		ctx context.Context,
		provider string,
		redirectURI string,
		state string,
		nonce string,
		loginHint string,
		codeChallenge string,
	) (string, error)
	ExchangeCode(
		ctx context.Context,
		provider string,
		code string,
		redirectURI string,
		nonce string,
		codeVerifier string,
	) (OIDCIdentity, error)
	RefreshToken(ctx context.Context, provider string, refreshToken string) (OIDCTokenSet, error)
}

// OIDCIdentity is the canonical identity result returned by OIDC verification.
// It is intentionally provider-neutral so application logic does not branch on adapter types.
type OIDCIdentity struct {
	Provider      string
	Issuer        string
	Subject       string
	ProviderSub   string
	Email         string
	EmailVerified bool
	Name          string
	AccessToken   string
	RefreshToken  string
	ExpiresAt     *time.Time
}

// OIDCTokenSet is the normalized token payload returned by refresh grant calls.
type OIDCTokenSet struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}
