package security

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

type OIDCProviderConfig struct {
	IssuerURL             string
	DiscoveryURL          string
	AuthorizationEndpoint string
	TokenEndpoint         string
	JWKSURI               string
	ClientID              string
	ClientSecret          string
	Scopes                []string
}

type OIDCVerifierConfig struct {
	HTTPClient *http.Client
	Providers  map[string]OIDCProviderConfig
}

type OIDCVerifier struct {
	httpClient *http.Client
	providers  map[string]OIDCProviderConfig
}

type oidcDiscoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewOIDCVerifier(cfg OIDCVerifierConfig) *OIDCVerifier {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	providers := make(map[string]OIDCProviderConfig, len(cfg.Providers))
	for name, provider := range cfg.Providers {
		providers[strings.ToLower(strings.TrimSpace(name))] = provider
	}
	return &OIDCVerifier{
		httpClient: httpClient,
		providers:  providers,
	}
}

func (v *OIDCVerifier) BuildAuthorizeURL(
	ctx context.Context,
	provider string,
	redirectURI string,
	state string,
	nonce string,
	loginHint string,
	codeChallenge string,
) (string, error) {
	providerCfg, err := v.providerConfig(provider)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(redirectURI) == "" || strings.TrimSpace(state) == "" || strings.TrimSpace(nonce) == "" {
		return "", fmt.Errorf("redirect_uri, state and nonce are required")
	}
	if _, err := url.ParseRequestURI(redirectURI); err != nil {
		return "", fmt.Errorf("invalid redirect_uri: %w", err)
	}

	discovery, err := v.discover(ctx, providerCfg)
	if err != nil {
		return "", err
	}

	q := url.Values{}
	q.Set("client_id", providerCfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(scopesOrDefault(providerCfg.Scopes), " "))
	q.Set("state", state)
	q.Set("nonce", nonce)
	if strings.TrimSpace(loginHint) != "" {
		q.Set("login_hint", strings.TrimSpace(loginHint))
	}
	if strings.TrimSpace(codeChallenge) != "" {
		q.Set("code_challenge", strings.TrimSpace(codeChallenge))
		q.Set("code_challenge_method", "S256")
	}

	return discovery.AuthorizationEndpoint + "?" + q.Encode(), nil
}

func (v *OIDCVerifier) ExchangeCode(
	ctx context.Context,
	provider string,
	code string,
	redirectURI string,
	nonce string,
	codeVerifier string,
) (ports.OIDCIdentity, error) {
	providerCfg, err := v.providerConfig(provider)
	if err != nil {
		return ports.OIDCIdentity{}, err
	}
	if strings.TrimSpace(code) == "" {
		return ports.OIDCIdentity{}, fmt.Errorf("authorization code is required")
	}

	discovery, err := v.discover(ctx, providerCfg)
	if err != nil {
		return ports.OIDCIdentity{}, err
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", providerCfg.ClientID)
	if strings.TrimSpace(providerCfg.ClientSecret) != "" {
		form.Set("client_secret", providerCfg.ClientSecret)
	}
	if strings.TrimSpace(redirectURI) != "" {
		form.Set("redirect_uri", redirectURI)
	}
	if strings.TrimSpace(codeVerifier) != "" {
		form.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return ports.OIDCIdentity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return ports.OIDCIdentity{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ports.OIDCIdentity{}, fmt.Errorf("oidc token exchange failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp oidcTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return ports.OIDCIdentity{}, fmt.Errorf("decode token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.IDToken) == "" {
		return ports.OIDCIdentity{}, fmt.Errorf("id_token missing in token response")
	}

	keySet, err := v.fetchJWKS(ctx, discovery.JWKSURI)
	if err != nil {
		return ports.OIDCIdentity{}, err
	}

	identity, err := validateIDToken(tokenResp.IDToken, keySet, discovery.Issuer, providerCfg.ClientID, strings.TrimSpace(nonce))
	if err != nil {
		return ports.OIDCIdentity{}, err
	}
	identity.Provider = strings.ToLower(strings.TrimSpace(provider))
	return identity, nil
}

func (v *OIDCVerifier) providerConfig(provider string) (OIDCProviderConfig, error) {
	name := strings.ToLower(strings.TrimSpace(provider))
	cfg, ok := v.providers[name]
	if !ok {
		return OIDCProviderConfig{}, fmt.Errorf("unsupported provider: %s", provider)
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return OIDCProviderConfig{}, fmt.Errorf("provider %s is not configured (missing client_id)", provider)
	}
	if strings.TrimSpace(cfg.IssuerURL) == "" && strings.TrimSpace(cfg.DiscoveryURL) == "" {
		return OIDCProviderConfig{}, fmt.Errorf("provider %s is not configured (missing issuer_url/discovery_url)", provider)
	}
	return cfg, nil
}

func (v *OIDCVerifier) discover(ctx context.Context, providerCfg OIDCProviderConfig) (oidcDiscoveryDocument, error) {
	discoveryURL := strings.TrimSpace(providerCfg.DiscoveryURL)
	if discoveryURL == "" {
		discoveryURL = strings.TrimRight(strings.TrimSpace(providerCfg.IssuerURL), "/") + "/.well-known/openid-configuration"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return oidcDiscoveryDocument{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return oidcDiscoveryDocument{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return oidcDiscoveryDocument{}, fmt.Errorf("oidc discovery failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var doc oidcDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("decode discovery document: %w", err)
	}

	if strings.TrimSpace(doc.Issuer) == "" {
		doc.Issuer = strings.TrimSpace(providerCfg.IssuerURL)
	}
	if strings.TrimSpace(providerCfg.IssuerURL) != "" && strings.TrimSpace(doc.Issuer) != strings.TrimSpace(providerCfg.IssuerURL) {
		return oidcDiscoveryDocument{}, fmt.Errorf("issuer mismatch: got %s expected %s", doc.Issuer, providerCfg.IssuerURL)
	}
	if strings.TrimSpace(doc.AuthorizationEndpoint) == "" {
		doc.AuthorizationEndpoint = strings.TrimSpace(providerCfg.AuthorizationEndpoint)
	}
	if strings.TrimSpace(doc.TokenEndpoint) == "" {
		doc.TokenEndpoint = strings.TrimSpace(providerCfg.TokenEndpoint)
	}
	if strings.TrimSpace(doc.JWKSURI) == "" {
		doc.JWKSURI = strings.TrimSpace(providerCfg.JWKSURI)
	}
	if strings.TrimSpace(doc.AuthorizationEndpoint) == "" || strings.TrimSpace(doc.TokenEndpoint) == "" || strings.TrimSpace(doc.JWKSURI) == "" {
		return oidcDiscoveryDocument{}, fmt.Errorf("discovery document missing required endpoints")
	}
	return doc, nil
}

func (v *OIDCVerifier) fetchJWKS(ctx context.Context, jwksURI string) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("oidc jwks fetch failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey)
	for i, key := range doc.Keys {
		if strings.ToUpper(strings.TrimSpace(key.Kty)) != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(key.N))
		if err != nil {
			return nil, fmt.Errorf("decode jwks n: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(key.E))
		if err != nil {
			return nil, fmt.Errorf("decode jwks e: %w", err)
		}
		eBig := new(big.Int).SetBytes(eBytes)
		if !eBig.IsInt64() {
			return nil, fmt.Errorf("invalid jwks exponent for key %s", key.Kid)
		}
		eValue := int(eBig.Int64())
		if eValue <= 1 {
			return nil, fmt.Errorf("invalid jwks exponent for key %s", key.Kid)
		}

		kid := strings.TrimSpace(key.Kid)
		if kid == "" {
			kid = fmt.Sprintf("key-%d", i)
		}
		keys[kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: eValue,
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no RSA keys found in jwks")
	}
	return keys, nil
}

func validateIDToken(raw string, keySet map[string]*rsa.PublicKey, issuer, clientID, expectedNonce string) (ports.OIDCIdentity, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			kid, _ := token.Header["kid"].(string)
			if strings.TrimSpace(kid) != "" {
				key, ok := keySet[kid]
				if !ok {
					return nil, fmt.Errorf("unknown key id: %s", kid)
				}
				return key, nil
			}
			if len(keySet) == 1 {
				for _, key := range keySet {
					return key, nil
				}
			}
			return nil, fmt.Errorf("missing key id")
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithAudience(clientID),
		jwt.WithIssuer(issuer),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return ports.OIDCIdentity{}, fmt.Errorf("validate id_token: %w", err)
	}
	if !parsed.Valid {
		return ports.OIDCIdentity{}, fmt.Errorf("invalid id_token")
	}

	subject := stringClaim(claims, "sub")
	if strings.TrimSpace(subject) == "" {
		return ports.OIDCIdentity{}, fmt.Errorf("id_token missing sub")
	}
	nonce := stringClaim(claims, "nonce")
	if strings.TrimSpace(expectedNonce) != "" && strings.TrimSpace(nonce) != strings.TrimSpace(expectedNonce) {
		return ports.OIDCIdentity{}, fmt.Errorf("nonce mismatch")
	}

	return ports.OIDCIdentity{
		ProviderSub:   subject,
		Email:         strings.ToLower(strings.TrimSpace(stringClaim(claims, "email"))),
		EmailVerified: boolClaim(claims["email_verified"]),
		Name:          strings.TrimSpace(stringClaim(claims, "name")),
	}, nil
}

func stringClaim(claims jwt.MapClaims, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func boolClaim(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func scopesOrDefault(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{"openid", "email", "profile"}
	}
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return out
}
