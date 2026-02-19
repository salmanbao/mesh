package unit

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

func TestRegisterLoginRefreshLogout(t *testing.T) {
	t.Parallel()

	f := newFixture()
	ctx := context.Background()

	registerRes, err := f.service.Register(ctx, application.RegisterRequest{
		Email:         "user@example.com",
		Password:      "SecurePass123!",
		Role:          "EDITOR",
		TermsAccepted: true,
	}, "idem-1")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registerRes.UserID == uuid.Nil {
		t.Fatalf("register returned empty user id")
	}

	loginRes, err := f.service.Login(ctx, application.LoginRequest{
		Email:      "user@example.com",
		Password:   "SecurePass123!",
		IPAddress:  "127.0.0.1",
		UserAgent:  "unit-test",
		DeviceName: "test",
		DeviceOS:   "linux",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginRes.Token == "" {
		t.Fatalf("login token should not be empty")
	}

	refreshRes, err := f.service.Refresh(ctx, loginRes.Token)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshRes.Token == "" {
		t.Fatalf("refresh token should not be empty")
	}

	if err := f.service.LogoutCurrentSession(ctx, loginRes.Token); err != nil {
		t.Fatalf("logout current session failed: %v", err)
	}
	if _, err := f.service.Refresh(ctx, loginRes.Token); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("expected revoked session after logout, got %v", err)
	}
}

func TestLoginWith2FAChallengeAndVerify(t *testing.T) {
	t.Parallel()

	f := newFixture()
	ctx := context.Background()

	registerRes, err := f.service.Register(ctx, application.RegisterRequest{
		Email:         "mfa@example.com",
		Password:      "SecurePass123!",
		Role:          "EDITOR",
		TermsAccepted: true,
	}, "")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := f.mfa.SetMethodEnabled(ctx, registerRes.UserID, "email", true, true, time.Now().UTC()); err != nil {
		t.Fatalf("enable 2fa failed: %v", err)
	}

	loginRes, err := f.service.Login(ctx, application.LoginRequest{
		Email:      "mfa@example.com",
		Password:   "SecurePass123!",
		IPAddress:  "127.0.0.1",
		UserAgent:  "unit-test",
		DeviceName: "test",
		DeviceOS:   "linux",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if !loginRes.Requires2FA || loginRes.TempToken == "" {
		t.Fatalf("expected temp token with requires_2fa")
	}

	challenge, err := f.challenges.Get(ctx, loginRes.TempToken)
	if err != nil || challenge == nil {
		t.Fatalf("expected persisted 2fa challenge")
	}
	verifyRes, err := f.service.Verify2FA(ctx, application.TwoFAVerifyRequest{
		TempToken: loginRes.TempToken,
		Code:      challenge.Code,
		Method:    "email",
	})
	if err != nil {
		t.Fatalf("verify 2fa failed: %v", err)
	}
	if verifyRes.Token == "" {
		t.Fatalf("expected jwt after 2fa verify")
	}
}

func TestOIDCAuthorizeAndCallback(t *testing.T) {
	t.Parallel()

	f := newFixture()
	ctx := context.Background()

	const (
		redirectURI = "https://app.example.com/auth/callback"
		state       = "state-123"
		nonce       = "nonce-123"
	)

	authURL, err := f.service.OIDCAuthorize(ctx, "google", redirectURI, state, nonce, "oidc@example.com")
	if err != nil {
		t.Fatalf("oidc authorize failed: %v", err)
	}
	if !strings.Contains(authURL, "state=state-123") {
		t.Fatalf("expected state in authorize url, got: %s", authURL)
	}

	callbackURL, err := f.service.OIDCCallback(ctx, "code-ok", state)
	if err != nil {
		t.Fatalf("oidc callback failed: %v", err)
	}
	u, err := url.Parse(callbackURL)
	if err != nil {
		t.Fatalf("parse callback redirect: %v", err)
	}
	if u.Fragment == "" || !strings.Contains(u.Fragment, "token=") {
		t.Fatalf("expected token fragment in callback redirect, got: %s", callbackURL)
	}
}

func newFixture() *fixture {
	users := &fakeUsers{
		byEmail: make(map[string]domain.User),
		byID:    make(map[uuid.UUID]domain.User),
	}
	sessions := &fakeSessions{byID: make(map[uuid.UUID]domain.Session)}
	loginAttempts := &fakeLoginAttempts{}
	idem := &fakeIdempotency{records: map[string]ports.IdempotencyRecord{}}
	lockouts := &fakeLockouts{state: map[string]ports.LockoutState{}}
	revocations := &fakeRevocations{revoked: map[uuid.UUID]bool{}}
	recovery := &fakeRecovery{}
	credentials := &fakeCredentials{users: users}
	mfa := &fakeMFA{methods: map[uuid.UUID][]string{}}
	oidc := &fakeOIDC{byProviderSubject: map[string]uuid.UUID{}, connections: map[uuid.UUID]map[string]bool{}}
	oidcVerifier := &fakeOIDCVerifier{
		identities: map[string]ports.OIDCIdentity{
			"code-ok": {
				Provider:      "google",
				ProviderSub:   "provider-sub-1",
				Email:         "oidc@example.com",
				EmailVerified: true,
				Name:          "OIDC User",
			},
		},
	}
	challenges := &fakeChallenges{items: map[string]ports.MFAChallenge{}}
	oidcStates := &fakeOIDCStateStore{items: map[string]ports.OIDCAuthState{}}
	signer := &fakeSigner{tokens: map[string]ports.AuthClaims{}}

	svc := application.NewService(application.Dependencies{
		Config: application.Config{
			DefaultRole:          "INFLUENCER",
			TokenTTL:             24 * time.Hour,
			SessionTTL:           30 * 24 * time.Hour,
			SessionAbsoluteTTL:   90 * 24 * time.Hour,
			FailedLoginThreshold: 5,
			LockoutDuration:      30 * time.Minute,
		},
		Users:         users,
		Sessions:      sessions,
		LoginAttempts: loginAttempts,
		Outbox:        &fakeOutbox{},
		Idempotency:   idem,
		Recovery:      recovery,
		Credentials:   credentials,
		MFA:           mfa,
		OIDC:          oidc,
		Lockouts:      lockouts,
		Revocations:   revocations,
		Challenges:    challenges,
		OIDCState:     oidcStates,
		OIDCVerifier:  oidcVerifier,
		Hasher:        &fakeHasher{},
		TokenSigner:   signer,
	})

	return &fixture{
		service:    svc,
		users:      users,
		mfa:        mfa,
		challenges: challenges,
	}
}

type fixture struct {
	service    *application.Service
	users      *fakeUsers
	mfa        *fakeMFA
	challenges *fakeChallenges
}

type fakeUsers struct {
	mu      sync.Mutex
	byEmail map[string]domain.User
	byID    map[uuid.UUID]domain.User
}

func (f *fakeUsers) CreateWithOutboxTx(_ context.Context, params ports.CreateUserTxParams, _ ports.OutboxEvent) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byEmail[params.Email]; ok {
		return domain.User{}, domain.ErrConflict
	}
	u := domain.User{
		UserID:       uuid.New(),
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		RoleName:     params.RoleName,
		IsActive:     true,
		CreatedAt:    params.RegisteredAtUTC,
		UpdatedAt:    params.RegisteredAtUTC,
	}
	f.byEmail[u.Email] = u
	f.byID[u.UserID] = u
	return u, nil
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) GetByID(_ context.Context, userID uuid.UUID) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[userID]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

type fakeSessions struct {
	mu   sync.Mutex
	byID map[uuid.UUID]domain.Session
}

func (f *fakeSessions) Create(_ context.Context, params ports.SessionCreateParams) (domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.byID[s.SessionID] = s
	return s, nil
}

func (f *fakeSessions) GetByID(_ context.Context, sessionID uuid.UUID) (domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[sessionID]
	if !ok {
		return domain.Session{}, domain.ErrNotFound
	}
	return s, nil
}

func (f *fakeSessions) ListByUser(_ context.Context, userID uuid.UUID, _, _ int) ([]domain.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Session
	for _, s := range f.byID {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSessions) TouchActivity(_ context.Context, sessionID uuid.UUID, touchedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.byID[sessionID]
	s.LastActivityAt = touchedAt
	f.byID[sessionID] = s
	return nil
}

func (f *fakeSessions) RevokeByID(_ context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byID[sessionID]
	if !ok {
		return domain.ErrNotFound
	}
	s.RevokedAt = &revokedAt
	f.byID[sessionID] = s
	return nil
}

func (f *fakeSessions) RevokeAllByUser(_ context.Context, userID uuid.UUID, revokedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, s := range f.byID {
		if s.UserID == userID {
			s.RevokedAt = &revokedAt
			f.byID[k] = s
		}
	}
	return nil
}

type fakeLoginAttempts struct{}

func (f *fakeLoginAttempts) Insert(context.Context, domain.LoginAttempt) error { return nil }

func (f *fakeLoginAttempts) ListByUser(context.Context, uuid.UUID, int, int, *time.Time, string) ([]domain.LoginAttempt, error) {
	return nil, nil
}

type fakeOutbox struct{}

func (f *fakeOutbox) Enqueue(context.Context, ports.OutboxEvent) error { return nil }
func (f *fakeOutbox) FetchUnpublished(context.Context, int) ([]ports.OutboxRecord, error) {
	return nil, nil
}
func (f *fakeOutbox) MarkPublished(context.Context, uuid.UUID, time.Time) error { return nil }
func (f *fakeOutbox) MarkFailed(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}

type fakeIdempotency struct {
	mu      sync.Mutex
	records map[string]ports.IdempotencyRecord
}

func (f *fakeIdempotency) Get(_ context.Context, key string) (*ports.IdempotencyRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.records[key]
	if !ok {
		return nil, nil
	}
	cp := v
	return &cp, nil
}

func (f *fakeIdempotency) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.records[key]; ok {
		return domain.ErrConflict
	}
	f.records[key] = ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Status:      "PENDING",
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	return nil
}

func (f *fakeIdempotency) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	v := f.records[key]
	v.Status = "COMPLETED"
	v.ResponseCode = responseCode
	v.ResponseBody = responseBody
	v.UpdatedAt = at
	f.records[key] = v
	return nil
}

type fakeLockouts struct {
	mu    sync.Mutex
	state map[string]ports.LockoutState
}

func (f *fakeLockouts) Get(_ context.Context, key string) (ports.LockoutState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state[key], nil
}

func (f *fakeLockouts) RecordFailure(_ context.Context, key string, now time.Time, threshold int, lockoutWindow time.Duration) (ports.LockoutState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.state[key]
	st.FailedCount++
	if st.FailedCount >= threshold {
		lockUntil := now.Add(lockoutWindow)
		st.LockedUntil = &lockUntil
	}
	f.state[key] = st
	return st, nil
}

func (f *fakeLockouts) Clear(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.state, key)
	return nil
}

type fakeRevocations struct {
	mu      sync.Mutex
	revoked map[uuid.UUID]bool
}

func (f *fakeRevocations) MarkRevoked(_ context.Context, sessionID uuid.UUID, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked[sessionID] = true
	return nil
}

func (f *fakeRevocations) IsRevoked(_ context.Context, sessionID uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.revoked[sessionID], nil
}

type fakeRecovery struct{}

func (f *fakeRecovery) CreatePasswordResetToken(context.Context, uuid.UUID, string, time.Time, time.Time) error {
	return nil
}
func (f *fakeRecovery) ConsumePasswordResetToken(context.Context, string, time.Time) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (f *fakeRecovery) CreateEmailVerificationToken(context.Context, uuid.UUID, string, time.Time, time.Time) error {
	return nil
}
func (f *fakeRecovery) ConsumeEmailVerificationToken(context.Context, string, time.Time) (uuid.UUID, error) {
	return uuid.New(), nil
}

type fakeCredentials struct {
	users *fakeUsers
}

func (f *fakeCredentials) UpdatePassword(_ context.Context, userID uuid.UUID, passwordHash string, updatedAt time.Time) error {
	f.users.mu.Lock()
	defer f.users.mu.Unlock()
	u, ok := f.users.byID[userID]
	if !ok {
		return domain.ErrNotFound
	}
	u.PasswordHash = passwordHash
	u.UpdatedAt = updatedAt
	f.users.byID[userID] = u
	f.users.byEmail[u.Email] = u
	return nil
}

func (f *fakeCredentials) SetEmailVerified(_ context.Context, userID uuid.UUID, verified bool, updatedAt time.Time) error {
	f.users.mu.Lock()
	defer f.users.mu.Unlock()
	u, ok := f.users.byID[userID]
	if !ok {
		return domain.ErrNotFound
	}
	u.EmailVerified = verified
	u.UpdatedAt = updatedAt
	f.users.byID[userID] = u
	f.users.byEmail[u.Email] = u
	return nil
}

func (f *fakeCredentials) HasPassword(_ context.Context, userID uuid.UUID) (bool, error) {
	f.users.mu.Lock()
	defer f.users.mu.Unlock()
	u, ok := f.users.byID[userID]
	if !ok {
		return false, domain.ErrNotFound
	}
	return u.PasswordHash != "", nil
}

type fakeMFA struct {
	mu      sync.Mutex
	methods map[uuid.UUID][]string
}

func (f *fakeMFA) ListEnabledMethods(_ context.Context, userID uuid.UUID) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]string{}, f.methods[userID]...)
	return out, nil
}

func (f *fakeMFA) SetMethodEnabled(_ context.Context, userID uuid.UUID, method string, enabled bool, _ bool, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	current := f.methods[userID]
	if enabled {
		found := false
		for _, m := range current {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			current = append(current, method)
		}
	} else {
		next := make([]string, 0, len(current))
		for _, m := range current {
			if m != method {
				next = append(next, m)
			}
		}
		current = next
	}
	f.methods[userID] = current
	return nil
}

func (f *fakeMFA) UpsertTOTPSecret(context.Context, uuid.UUID, []byte, time.Time) error { return nil }
func (f *fakeMFA) ReplaceBackupCodes(context.Context, uuid.UUID, []string, time.Time) error {
	return nil
}
func (f *fakeMFA) ConsumeBackupCode(context.Context, uuid.UUID, string, time.Time) (bool, error) {
	return false, nil
}

type fakeOIDC struct {
	mu                sync.Mutex
	byProviderSubject map[string]uuid.UUID
	connections       map[uuid.UUID]map[string]bool
}

func (f *fakeOIDC) FindUserByProviderSubject(_ context.Context, provider, providerUserID string) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := provider + ":" + providerUserID
	id, ok := f.byProviderSubject[key]
	if !ok {
		return uuid.Nil, domain.ErrNotFound
	}
	return id, nil
}

func (f *fakeOIDC) UpsertConnection(_ context.Context, userID uuid.UUID, provider, providerUserID, _ string, _ bool, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := provider + ":" + providerUserID
	f.byProviderSubject[key] = userID
	if f.connections[userID] == nil {
		f.connections[userID] = map[string]bool{}
	}
	f.connections[userID][provider] = true
	return nil
}

func (f *fakeOIDC) CountConnections(_ context.Context, userID uuid.UUID) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.connections[userID]), nil
}

func (f *fakeOIDC) DeleteConnection(_ context.Context, userID uuid.UUID, provider string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.connections[userID] == nil || !f.connections[userID][provider] {
		return false, nil
	}
	delete(f.connections[userID], provider)
	return true, nil
}

type fakeChallenges struct {
	mu    sync.Mutex
	items map[string]ports.MFAChallenge
}

func (f *fakeChallenges) Put(_ context.Context, token string, challenge ports.MFAChallenge, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items[token] = challenge
	return nil
}

func (f *fakeChallenges) Get(_ context.Context, token string) (*ports.MFAChallenge, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[token]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}

func (f *fakeChallenges) Delete(_ context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.items, token)
	return nil
}

type fakeOIDCStateStore struct {
	mu    sync.Mutex
	items map[string]ports.OIDCAuthState
}

func (f *fakeOIDCStateStore) Put(_ context.Context, state string, value ports.OIDCAuthState, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items[state] = value
	return nil
}

func (f *fakeOIDCStateStore) Get(_ context.Context, state string) (*ports.OIDCAuthState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	item, ok := f.items[state]
	if !ok {
		return nil, nil
	}
	cp := item
	return &cp, nil
}

func (f *fakeOIDCStateStore) Delete(_ context.Context, state string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.items, state)
	return nil
}

type fakeOIDCVerifier struct {
	mu         sync.Mutex
	identities map[string]ports.OIDCIdentity
}

func (f *fakeOIDCVerifier) BuildAuthorizeURL(_ context.Context, provider, redirectURI, state, nonce, loginHint, codeChallenge string) (string, error) {
	q := url.Values{}
	q.Set("provider", provider)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("login_hint", loginHint)
	q.Set("code_challenge", codeChallenge)
	return "https://idp.example.test/auth?" + q.Encode(), nil
}

func (f *fakeOIDCVerifier) ExchangeCode(_ context.Context, provider, code, _, _, _ string) (ports.OIDCIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	identity, ok := f.identities[code]
	if !ok {
		return ports.OIDCIdentity{}, domain.ErrUnauthorized
	}
	identity.Provider = provider
	return identity, nil
}

type fakeHasher struct{}

func (f *fakeHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

func (f *fakeHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return errors.New("hash mismatch")
	}
	return nil
}

type fakeSigner struct {
	mu     sync.Mutex
	tokens map[string]ports.AuthClaims
}

func (f *fakeSigner) Sign(claims ports.AuthClaims) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	token := uuid.NewString()
	f.tokens[token] = claims
	return token, nil
}

func (f *fakeSigner) ParseAndValidate(token string) (ports.AuthClaims, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	claims, ok := f.tokens[token]
	if !ok {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	return claims, nil
}

func (f *fakeSigner) PublicJWKs() ([]map[string]any, error) {
	return []map[string]any{{"kid": "fake"}}, nil
}
