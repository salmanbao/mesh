package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type AuthClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	SessionID uuid.UUID `json:"session_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	KeyID     string    `json:"kid"`
}

type TokenSigner interface {
	Sign(claims AuthClaims) (string, error)
	ParseAndValidate(token string) (AuthClaims, error)
	PublicJWKs() ([]map[string]any, error)
}

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
}

type OIDCIdentity struct {
	Provider      string
	ProviderSub   string
	Email         string
	EmailVerified bool
	Name          string
}
