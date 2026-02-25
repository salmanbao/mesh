package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// JWTSigner implements RS256 token signing/parsing for M01 auth sessions.
// Keys are held at adapter level so application layer stays crypto-library agnostic.
type JWTSigner struct {
	kid        string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewJWTSigner builds a signer from configured PEM keys.
func NewJWTSigner(kid, privateKeyPEM, publicKeyPEM string) (*JWTSigner, error) {
	if kid == "" {
		return nil, errors.New("jwt key id (kid) is required")
	}
	if privateKeyPEM == "" || publicKeyPEM == "" {
		return nil, errors.New("jwt private/public keys are required")
	}

	priv, err := parseRSAPrivate(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	pub, err := parseRSAPublic(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &JWTSigner{
		kid:        kid,
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// NewEphemeralJWTSigner creates an in-memory keypair for local/dev use.
// This exists to unblock runtime startup when static keys are intentionally absent.
func NewEphemeralJWTSigner(kid string) (*JWTSigner, error) {
	if kid == "" {
		kid = "ephemeral-key-1"
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &JWTSigner{
		kid:        kid,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

type authJWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func (s *JWTSigner) Sign(claims ports.AuthClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, authJWTClaims{
		UserID:    claims.UserID.String(),
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: claims.SessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(claims.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(claims.ExpiresAt),
		},
	})
	token.Header["kid"] = s.kid
	return token.SignedString(s.privateKey)
}

func (s *JWTSigner) ParseAndValidate(raw string) (ports.AuthClaims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &authJWTClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return s.publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithLeeway(30*time.Second))
	if err != nil {
		return ports.AuthClaims{}, err
	}
	claims, ok := parsed.Claims.(*authJWTClaims)
	if !ok || !parsed.Valid {
		return ports.AuthClaims{}, errors.New("invalid token claims")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return ports.AuthClaims{}, fmt.Errorf("parse user_id: %w", err)
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return ports.AuthClaims{}, fmt.Errorf("parse session_id: %w", err)
	}

	kid, _ := parsed.Header["kid"].(string)

	return ports.AuthClaims{
		UserID:    userID,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: sessionID,
		IssuedAt:  claims.IssuedAt.Time.UTC(),
		ExpiresAt: claims.ExpiresAt.Time.UTC(),
		KeyID:     kid,
	}, nil
}

func (s *JWTSigner) PublicJWKs() ([]map[string]any, error) {
	e := big.NewInt(int64(s.publicKey.E)).Bytes()
	n := s.publicKey.N.Bytes()

	return []map[string]any{
		{
			"kid": s.kid,
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(n),
			"e":   base64.RawURLEncoding.EncodeToString(e),
		},
	}, nil
}

func parseRSAPrivate(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("invalid private PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func parseRSAPublic(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("invalid public PEM")
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := keyAny.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return key, nil
}
