package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/security"
)

func TestOIDCVerifier_RealDiscoveryJWKSAndExchange(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	const kid = "test-kid-1"

	var issuerURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuerURL,
			"authorization_endpoint": issuerURL + "/oauth2/v2/auth",
			"token_endpoint":         issuerURL + "/oauth2/v2/token",
			"jwks_uri":               issuerURL + "/oauth2/v3/certs",
		})
	})
	mux.HandleFunc("/oauth2/v3/certs", func(w http.ResponseWriter, r *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(bigEndianInt(privateKey.PublicKey.E))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   n,
					"e":   e,
				},
			},
		})
	})
	mux.HandleFunc("/oauth2/v2/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("code") == "" || r.Form.Get("client_id") == "" {
			http.Error(w, "missing code/client_id", http.StatusBadRequest)
			return
		}
		now := time.Now().UTC()
		idToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss":            issuerURL,
			"aud":            "mesh-test-client",
			"sub":            "provider-sub-123",
			"email":          "oidc-user@example.com",
			"email_verified": true,
			"name":           "OIDC User",
			"nonce":          "nonce-123",
			"iat":            now.Unix(),
			"exp":            now.Add(5 * time.Minute).Unix(),
		})
		idToken.Header["kid"] = kid
		signed, err := idToken.SignedString(privateKey)
		if err != nil {
			http.Error(w, "sign token failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token",
			"id_token":     signed,
			"token_type":   "Bearer",
			"expires_in":   300,
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	issuerURL = server.URL

	verifier := security.NewOIDCVerifier(security.OIDCVerifierConfig{
		HTTPClient: server.Client(),
		Providers: map[string]security.OIDCProviderConfig{
			"google": {
				IssuerURL:    issuerURL,
				ClientID:     "mesh-test-client",
				ClientSecret: "mesh-test-secret",
				Scopes:       []string{"openid", "email", "profile"},
			},
		},
	})

	authURL, err := verifier.BuildAuthorizeURL(
		context.Background(),
		"google",
		"https://app.example.com/auth/callback",
		"state-123",
		"nonce-123",
		"user@example.com",
		"pkce-challenge",
	)
	if err != nil {
		t.Fatalf("build authorize url: %v", err)
	}
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	if got := u.Query().Get("code_challenge_method"); got != "S256" {
		t.Fatalf("expected code_challenge_method=S256, got %s", got)
	}
	if got := u.Query().Get("nonce"); got != "nonce-123" {
		t.Fatalf("expected nonce in authorize url, got %s", got)
	}

	identity, err := verifier.ExchangeCode(
		context.Background(),
		"google",
		"auth-code-1",
		"https://app.example.com/auth/callback",
		"nonce-123",
		"pkce-verifier",
	)
	if err != nil {
		t.Fatalf("exchange code failed: %v", err)
	}
	if identity.ProviderSub != "provider-sub-123" {
		t.Fatalf("unexpected provider sub: %s", identity.ProviderSub)
	}
	if !identity.EmailVerified || !strings.EqualFold(identity.Email, "oidc-user@example.com") {
		t.Fatalf("unexpected identity payload: %+v", identity)
	}

	_, err = verifier.ExchangeCode(
		context.Background(),
		"google",
		"auth-code-1",
		"https://app.example.com/auth/callback",
		"nonce-mismatch",
		"pkce-verifier",
	)
	if err == nil {
		t.Fatalf("expected nonce mismatch error")
	}
}

func bigEndianInt(v int) []byte {
	if v <= 0 {
		return []byte{0}
	}
	out := make([]byte, 0, 4)
	for n := v; n > 0; n >>= 8 {
		out = append([]byte{byte(n & 0xff)}, out...)
	}
	return out
}
