package contract

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	httpadapter "github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

func TestOIDCAuthorizeAndCallbackHTTPContract(t *testing.T) {
	t.Parallel()

	svc := newOIDCContractService()
	router := httpadapter.NewRouter(httpadapter.NewHandler(svc))

	authorizeReq := httptest.NewRequest(http.MethodGet, "/auth/v1/oidc/authorize?provider=google&redirect_uri=https://app.example.com/auth/callback&state=state-1&nonce=nonce-1", nil)
	authorizeRes := httptest.NewRecorder()
	router.ServeHTTP(authorizeRes, authorizeReq)
	if authorizeRes.Code != http.StatusFound {
		t.Fatalf("expected 302 authorize response, got %d", authorizeRes.Code)
	}
	location := authorizeRes.Header().Get("Location")
	if !strings.Contains(location, "state=state-1") {
		t.Fatalf("expected state in authorize location, got %s", location)
	}

	callbackReq := httptest.NewRequest(http.MethodGet, "/auth/v1/oidc/callback?code=code-ok&state=state-1", nil)
	callbackRes := httptest.NewRecorder()
	router.ServeHTTP(callbackRes, callbackReq)
	if callbackRes.Code != http.StatusFound {
		t.Fatalf("expected 302 callback response, got %d", callbackRes.Code)
	}
	callbackLocation := callbackRes.Header().Get("Location")
	parsed, err := url.Parse(callbackLocation)
	if err != nil {
		t.Fatalf("parse callback location: %v", err)
	}
	if parsed.Fragment == "" || !strings.Contains(parsed.Fragment, "token=") {
		t.Fatalf("expected token fragment in callback location, got %s", callbackLocation)
	}
}

func TestOIDCCallbackRejectsUnknownState(t *testing.T) {
	t.Parallel()

	svc := newOIDCContractService()
	router := httpadapter.NewRouter(httpadapter.NewHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/auth/v1/oidc/callback?code=code-ok&state=missing", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing OIDC state, got %d", res.Code)
	}
}

func newOIDCContractService() *application.Service {
	users := &contractUsers{byEmail: map[string]domain.User{}, byID: map[uuid.UUID]domain.User{}}
	return application.NewService(application.Dependencies{
		Config: application.Config{
			DefaultRole:          "INFLUENCER",
			TokenTTL:             24 * time.Hour,
			SessionTTL:           30 * 24 * time.Hour,
			SessionAbsoluteTTL:   90 * 24 * time.Hour,
			FailedLoginThreshold: 5,
			LockoutDuration:      30 * time.Minute,
		},
		Users:         users,
		Sessions:      &contractSessions{items: map[uuid.UUID]domain.Session{}},
		LoginAttempts: noopLoginAttempts{},
		Outbox:        noopOutbox{},
		Idempotency:   noopIdempotency{},
		Recovery:      noopRecovery{},
		Credentials:   &contractCredentials{users: users},
		MFA:           noopMFA{},
		OIDC:          &contractOIDCRepo{connections: map[string]uuid.UUID{}},
		Lockouts:      noopLockouts{},
		Revocations:   noopRevocations{},
		Challenges:    noopChallengeStore{},
		OIDCState:     &contractOIDCStateStore{items: map[string]ports.OIDCAuthState{}},
		OIDCVerifier:  contractOIDCVerifier{},
		Hasher:        noopHasher{},
		TokenSigner:   &contractSigner{tokens: map[string]ports.AuthClaims{}},
	})
}

type contractOIDCVerifier struct{}

func (contractOIDCVerifier) BuildAuthorizeURL(_ context.Context, _ string, redirectURI, state, nonce, _ string, _ string) (string, error) {
	q := url.Values{}
	q.Set("state", state)
	q.Set("nonce", nonce)
	return "https://idp.example.test/auth?" + q.Encode() + "&redirect_uri=" + url.QueryEscape(redirectURI), nil
}

func (contractOIDCVerifier) ExchangeCode(_ context.Context, provider, code, _, _, _ string) (ports.OIDCIdentity, error) {
	if strings.TrimSpace(code) == "" {
		return ports.OIDCIdentity{}, errors.New("missing code")
	}
	return ports.OIDCIdentity{
		Provider:      provider,
		ProviderSub:   "provider-sub-" + code,
		Email:         "oidc@example.com",
		EmailVerified: true,
		Name:          "OIDC User",
	}, nil
}

type contractUsers struct {
	mu      sync.Mutex
	byEmail map[string]domain.User
	byID    map[uuid.UUID]domain.User
}

func (c *contractUsers) CreateWithOutboxTx(_ context.Context, params ports.CreateUserTxParams, _ ports.OutboxEvent) (domain.User, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u := domain.User{
		UserID:        uuid.New(),
		Email:         params.Email,
		PasswordHash:  params.PasswordHash,
		RoleName:      params.RoleName,
		EmailVerified: params.EmailVerified,
		CreatedAt:     params.RegisteredAtUTC,
		UpdatedAt:     params.RegisteredAtUTC,
		IsActive:      true,
	}
	c.byEmail[u.Email] = u
	c.byID[u.UserID] = u
	return u, nil
}

func (c *contractUsers) GetByEmail(_ context.Context, email string) (domain.User, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u, ok := c.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (c *contractUsers) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u, ok := c.byID[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

type contractSessions struct {
	mu    sync.Mutex
	items map[uuid.UUID]domain.Session
}

func (c *contractSessions) Create(_ context.Context, params ports.SessionCreateParams) (domain.Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := domain.Session{
		SessionID:      uuid.New(),
		UserID:         params.UserID,
		DeviceName:     params.DeviceName,
		DeviceOS:       params.DeviceOS,
		IPAddress:      params.IPAddress,
		UserAgent:      params.UserAgent,
		CreatedAt:      params.LastActivityAt,
		LastActivityAt: params.LastActivityAt,
		ExpiresAt:      params.ExpiresAt,
	}
	c.items[s.SessionID] = s
	return s, nil
}

func (c *contractSessions) GetByID(_ context.Context, sessionID uuid.UUID) (domain.Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.items[sessionID]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return s, nil
}

func (c *contractSessions) ListByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.Session, error) {
	return nil, nil
}
func (c *contractSessions) TouchActivity(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (c *contractSessions) RevokeByID(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (c *contractSessions) RevokeAllByUser(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

type contractOIDCRepo struct {
	mu          sync.Mutex
	connections map[string]uuid.UUID
}

func (c *contractOIDCRepo) FindUserByProviderSubject(_ context.Context, provider, providerUserID string) (uuid.UUID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id, ok := c.connections[provider+":"+providerUserID]
	if !ok {
		return uuid.Nil, domain.ErrNotFound
	}
	return id, nil
}

func (c *contractOIDCRepo) UpsertConnection(_ context.Context, userID uuid.UUID, provider, providerUserID, _ string, _ bool, _ time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connections[provider+":"+providerUserID] = userID
	return nil
}

func (c *contractOIDCRepo) CountConnections(_ context.Context, _ uuid.UUID) (int, error) {
	return 1, nil
}
func (c *contractOIDCRepo) DeleteConnection(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return true, nil
}

type contractOIDCStateStore struct {
	mu    sync.Mutex
	items map[string]ports.OIDCAuthState
}

func (c *contractOIDCStateStore) Put(_ context.Context, state string, value ports.OIDCAuthState, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[state] = value
	return nil
}

func (c *contractOIDCStateStore) Get(_ context.Context, state string) (*ports.OIDCAuthState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.items[state]
	if !ok {
		return nil, nil
	}
	cp := v
	return &cp, nil
}

func (c *contractOIDCStateStore) Delete(_ context.Context, state string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, state)
	return nil
}

type contractCredentials struct {
	users *contractUsers
}

func (c *contractCredentials) UpdatePassword(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (c *contractCredentials) SetEmailVerified(_ context.Context, _ uuid.UUID, _ bool, _ time.Time) error {
	return nil
}
func (c *contractCredentials) HasPassword(_ context.Context, _ uuid.UUID) (bool, error) {
	return true, nil
}

type contractSigner struct {
	mu     sync.Mutex
	tokens map[string]ports.AuthClaims
}

func (c *contractSigner) Sign(claims ports.AuthClaims) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	token := uuid.NewString()
	c.tokens[token] = claims
	return token, nil
}

func (c *contractSigner) ParseAndValidate(token string) (ports.AuthClaims, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	claims, ok := c.tokens[token]
	if !ok {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	return claims, nil
}

func (c *contractSigner) PublicJWKs() ([]map[string]any, error) {
	return []map[string]any{{"kid": "contract"}}, nil
}

type noopHasher struct{}

func (noopHasher) Hash(password string) (string, error) { return password, nil }
func (noopHasher) Compare(hash, password string) error  { return nil }

type noopLockouts struct{}

func (noopLockouts) Get(context.Context, string) (ports.LockoutState, error) {
	return ports.LockoutState{}, nil
}
func (noopLockouts) RecordFailure(context.Context, string, time.Time, int, time.Duration) (ports.LockoutState, error) {
	return ports.LockoutState{}, nil
}
func (noopLockouts) Clear(context.Context, string) error { return nil }

type noopRevocations struct{}

func (noopRevocations) MarkRevoked(context.Context, uuid.UUID, time.Time) error { return nil }
func (noopRevocations) IsRevoked(context.Context, uuid.UUID) (bool, error)      { return false, nil }

type noopChallengeStore struct{}

func (noopChallengeStore) Put(context.Context, string, ports.MFAChallenge, time.Duration) error {
	return nil
}
func (noopChallengeStore) Get(context.Context, string) (*ports.MFAChallenge, error) { return nil, nil }
func (noopChallengeStore) Delete(context.Context, string) error                     { return nil }

type noopLoginAttempts struct{}

func (noopLoginAttempts) Insert(context.Context, domain.LoginAttempt) error { return nil }
func (noopLoginAttempts) ListByUser(context.Context, uuid.UUID, int, int, *time.Time, string) ([]domain.LoginAttempt, error) {
	return nil, nil
}

type noopOutbox struct{}

func (noopOutbox) Enqueue(context.Context, ports.OutboxEvent) error { return nil }
func (noopOutbox) FetchUnpublished(context.Context, int) ([]ports.OutboxRecord, error) {
	return nil, nil
}
func (noopOutbox) MarkPublished(context.Context, uuid.UUID, time.Time) error { return nil }
func (noopOutbox) MarkFailed(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}

type noopIdempotency struct{}

func (noopIdempotency) Get(context.Context, string) (*ports.IdempotencyRecord, error) {
	return nil, nil
}
func (noopIdempotency) Reserve(context.Context, string, string, time.Time) error       { return nil }
func (noopIdempotency) Complete(context.Context, string, int, []byte, time.Time) error { return nil }

type noopRecovery struct{}

func (noopRecovery) CreatePasswordResetToken(context.Context, uuid.UUID, string, time.Time, time.Time) error {
	return nil
}
func (noopRecovery) ConsumePasswordResetToken(context.Context, string, time.Time) (uuid.UUID, error) {
	return uuid.Nil, domain.ErrNotFound
}
func (noopRecovery) CreateEmailVerificationToken(context.Context, uuid.UUID, string, time.Time, time.Time) error {
	return nil
}
func (noopRecovery) ConsumeEmailVerificationToken(context.Context, string, time.Time) (uuid.UUID, error) {
	return uuid.Nil, domain.ErrNotFound
}

type noopMFA struct{}

func (noopMFA) ListEnabledMethods(context.Context, uuid.UUID) ([]string, error) { return nil, nil }
func (noopMFA) SetMethodEnabled(context.Context, uuid.UUID, string, bool, bool, time.Time) error {
	return nil
}
func (noopMFA) UpsertTOTPSecret(context.Context, uuid.UUID, []byte, time.Time) error { return nil }
func (noopMFA) ReplaceBackupCodes(context.Context, uuid.UUID, []string, time.Time) error {
	return nil
}
func (noopMFA) ConsumeBackupCode(context.Context, uuid.UUID, string, time.Time) (bool, error) {
	return false, nil
}
