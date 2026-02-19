package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

type Service struct {
	cfg           Config
	users         ports.UserRepository
	sessions      ports.SessionRepository
	loginAttempts ports.LoginAttemptRepository
	outbox        ports.OutboxRepository
	idempotency   ports.IdempotencyRepository
	recovery      ports.RecoveryRepository
	credentials   ports.CredentialRepository
	mfa           ports.MFARepository
	oidc          ports.OIDCRepository
	lockouts      ports.LockoutStore
	revocations   ports.SessionRevocationStore
	challenges    ports.MFAChallengeStore
	oidcState     ports.OIDCStateStore
	oidcVerifier  ports.OIDCVerifier
	hasher        ports.PasswordHasher
	tokenSigner   ports.TokenSigner
	nowFn         func() time.Time
}

type Dependencies struct {
	Config        Config
	Users         ports.UserRepository
	Sessions      ports.SessionRepository
	LoginAttempts ports.LoginAttemptRepository
	Outbox        ports.OutboxRepository
	Idempotency   ports.IdempotencyRepository
	Recovery      ports.RecoveryRepository
	Credentials   ports.CredentialRepository
	MFA           ports.MFARepository
	OIDC          ports.OIDCRepository
	Lockouts      ports.LockoutStore
	Revocations   ports.SessionRevocationStore
	Challenges    ports.MFAChallengeStore
	OIDCState     ports.OIDCStateStore
	OIDCVerifier  ports.OIDCVerifier
	Hasher        ports.PasswordHasher
	TokenSigner   ports.TokenSigner
}

func NewService(deps Dependencies) *Service {
	return &Service{
		cfg:           deps.Config,
		users:         deps.Users,
		sessions:      deps.Sessions,
		loginAttempts: deps.LoginAttempts,
		outbox:        deps.Outbox,
		idempotency:   deps.Idempotency,
		recovery:      deps.Recovery,
		credentials:   deps.Credentials,
		mfa:           deps.MFA,
		oidc:          deps.OIDC,
		lockouts:      deps.Lockouts,
		revocations:   deps.Revocations,
		challenges:    deps.Challenges,
		oidcState:     deps.OIDCState,
		oidcVerifier:  deps.OIDCVerifier,
		hasher:        deps.Hasher,
		tokenSigner:   deps.TokenSigner,
		nowFn:         time.Now().UTC,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, idempotencyKey string) (RegisterResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return RegisterResponse{}, err
	}
	if !req.TermsAccepted {
		return RegisterResponse{}, fmt.Errorf("%w: terms must be accepted", domain.ErrInvalidInput)
	}

	if req.Provider != "" {
		return RegisterResponse{}, fmt.Errorf("%w: OIDC registration flow is tracked in later slice", domain.ErrNotImplemented)
	}

	if err := domain.ValidatePassword(req.Password); err != nil {
		return RegisterResponse{}, err
	}

	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role == "" {
		role = s.cfg.DefaultRole
	}

	if idempotencyKey != "" {
		requestHash := hashRequest(req)
		if err := s.idempotency.Reserve(ctx, idempotencyKey, requestHash, s.nowFn().Add(7*24*time.Hour)); err != nil {
			return RegisterResponse{}, fmt.Errorf("%w: %v", domain.ErrIdempotencyConflict, err)
		}
	}

	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("hash password: %w", err)
	}

	now := s.nowFn()
	payload, _ := json.Marshal(map[string]any{
		"user_id":       nil,
		"registered_at": now,
		"email":         email,
	})

	event := ports.OutboxEvent{
		EventID:      uuid.New(),
		EventType:    "user.registered",
		PartitionKey: email,
		Payload:      payload,
		OccurredAt:   now,
	}

	user, err := s.users.CreateWithOutboxTx(ctx, ports.CreateUserTxParams{
		Email:           email,
		PasswordHash:    passwordHash,
		RoleName:        role,
		EmailVerified:   false,
		IdempotencyKey:  idempotencyKey,
		RegisteredAtUTC: now,
	}, event)
	if err != nil {
		return RegisterResponse{}, err
	}

	if idempotencyKey != "" {
		responseBody, _ := json.Marshal(RegisterResponse{UserID: user.UserID})
		_ = s.idempotency.Complete(ctx, idempotencyKey, 201, responseBody, s.nowFn())
	}

	return RegisterResponse{UserID: user.UserID}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return LoginResponse{}, err
	}

	lockKey := "login:" + email
	lockState, err := s.lockouts.Get(ctx, lockKey)
	if err == nil && lockState.LockedUntil != nil && lockState.LockedUntil.After(s.nowFn()) {
		return LoginResponse{}, domain.ErrAccountLocked
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		s.recordFailure(ctx, nil, req, "USER_NOT_FOUND")
		return LoginResponse{}, domain.ErrInvalidCredentials
	}

	if err := s.hasher.Compare(user.PasswordHash, req.Password); err != nil {
		s.recordFailure(ctx, &user.UserID, req, "INVALID_PASSWORD")
		_, _ = s.lockouts.RecordFailure(ctx, lockKey, s.nowFn(), s.cfg.FailedLoginThreshold, s.cfg.LockoutDuration)
		return LoginResponse{}, domain.ErrInvalidCredentials
	}

	_ = s.lockouts.Clear(ctx, lockKey)

	now := s.nowFn()
	enabledMethods, err := s.mfa.ListEnabledMethods(ctx, user.UserID)
	if err == nil && len(enabledMethods) > 0 {
		method := enabledMethods[0]
		challengeCode := randomDigits(6)
		tempToken := uuid.NewString()
		expiresAt := now.Add(5 * time.Minute)
		if err := s.challenges.Put(ctx, tempToken, ports.MFAChallenge{
			UserID:    user.UserID,
			Email:     user.Email,
			Role:      user.RoleName,
			Method:    method,
			Code:      challengeCode,
			ExpiresAt: expiresAt,
		}, 5*time.Minute); err != nil {
			return LoginResponse{}, fmt.Errorf("store 2fa challenge: %w", err)
		}

		payload, _ := json.Marshal(map[string]any{
			"user_id":      user.UserID,
			"method":       method,
			"requested_at": now,
		})
		_ = s.outbox.Enqueue(ctx, ports.OutboxEvent{
			EventID:      uuid.New(),
			EventType:    "auth.2fa.required",
			PartitionKey: user.UserID.String(),
			Payload:      payload,
			OccurredAt:   now,
		})

		return LoginResponse{
			Requires2FA: true,
			TempToken:   tempToken,
		}, nil
	}

	session, err := s.sessions.Create(ctx, ports.SessionCreateParams{
		UserID:         user.UserID,
		DeviceName:     req.DeviceName,
		DeviceOS:       req.DeviceOS,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		ExpiresAt:      now.Add(s.cfg.SessionTTL),
		LastActivityAt: now,
	})
	if err != nil {
		return LoginResponse{}, fmt.Errorf("create session: %w", err)
	}

	_ = s.loginAttempts.Insert(ctx, domain.LoginAttempt{
		UserID:     &user.UserID,
		AttemptAt:  now,
		IPAddress:  req.IPAddress,
		Status:     "SUCCESS",
		DeviceName: req.DeviceName,
		DeviceOS:   req.DeviceOS,
		UserAgent:  req.UserAgent,
	})

	token, err := s.tokenSigner.Sign(ports.AuthClaims{
		UserID:    user.UserID,
		Email:     user.Email,
		Role:      user.RoleName,
		SessionID: session.SessionID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.cfg.TokenTTL),
	})
	if err != nil {
		return LoginResponse{}, fmt.Errorf("sign token: %w", err)
	}

	return LoginResponse{
		Requires2FA: false,
		Token:       token,
		SessionID:   session.SessionID,
		ExpiresIn:   int64(s.cfg.TokenTTL.Seconds()),
	}, nil
}

func (s *Service) Refresh(ctx context.Context, jwtToken string) (RefreshResponse, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return RefreshResponse{}, domain.ErrUnauthorized
	}

	session, err := s.sessions.GetByID(ctx, claims.SessionID)
	if err != nil {
		return RefreshResponse{}, domain.ErrUnauthorized
	}
	if session.RevokedAt != nil {
		return RefreshResponse{}, domain.ErrSessionRevoked
	}
	if session.ExpiresAt.Before(s.nowFn()) {
		return RefreshResponse{}, domain.ErrSessionExpired
	}
	if session.CreatedAt.Add(s.cfg.SessionAbsoluteTTL).Before(s.nowFn()) {
		return RefreshResponse{}, domain.ErrSessionExpired
	}
	if revoked, _ := s.revocations.IsRevoked(ctx, session.SessionID); revoked {
		return RefreshResponse{}, domain.ErrSessionRevoked
	}

	now := s.nowFn()
	_ = s.sessions.TouchActivity(ctx, session.SessionID, now)

	newToken, err := s.tokenSigner.Sign(ports.AuthClaims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: claims.SessionID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.cfg.TokenTTL),
	})
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("sign refreshed token: %w", err)
	}

	return RefreshResponse{
		Token:     newToken,
		ExpiresIn: int64(s.cfg.TokenTTL.Seconds()),
	}, nil
}

func (s *Service) LogoutCurrentSession(ctx context.Context, jwtToken string) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}
	now := s.nowFn()
	if err := s.sessions.RevokeByID(ctx, claims.SessionID, now); err != nil {
		return err
	}
	_ = s.revocations.MarkRevoked(ctx, claims.SessionID, now.Add(s.cfg.TokenTTL))
	return nil
}

func (s *Service) LogoutAllSessions(ctx context.Context, jwtToken string) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}
	now := s.nowFn()
	if err := s.sessions.RevokeAllByUser(ctx, claims.UserID, now); err != nil {
		return err
	}
	return nil
}

func (s *Service) RevokeSessionByID(ctx context.Context, jwtToken string, sessionID uuid.UUID) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}
	target, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return domain.ErrNotFound
	}
	if target.UserID != claims.UserID {
		return domain.ErrUnauthorized
	}

	now := s.nowFn()
	if err := s.sessions.RevokeByID(ctx, sessionID, now); err != nil {
		return err
	}
	_ = s.revocations.MarkRevoked(ctx, sessionID, now.Add(s.cfg.TokenTTL))
	return nil
}

func (s *Service) ListSessions(ctx context.Context, jwtToken string) ([]SessionItem, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	sessions, err := s.sessions.ListByUser(ctx, claims.UserID, 100, 0)
	if err != nil {
		return nil, err
	}

	result := make([]SessionItem, 0, len(sessions))
	for _, it := range sessions {
		result = append(result, toSessionItem(it, claims.SessionID))
	}
	return result, nil
}

func (s *Service) ListLoginHistory(ctx context.Context, jwtToken string, q LoginHistoryQuery) ([]LoginHistoryItem, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}
	offset := (q.Page - 1) * q.Limit

	var since *time.Time
	if q.Days > 0 {
		t := s.nowFn().Add(-time.Duration(q.Days) * 24 * time.Hour)
		since = &t
	}

	attempts, err := s.loginAttempts.ListByUser(ctx, claims.UserID, q.Limit, offset, since, strings.ToUpper(strings.TrimSpace(q.Status)))
	if err != nil {
		return nil, err
	}

	result := make([]LoginHistoryItem, 0, len(attempts))
	for _, attempt := range attempts {
		result = append(result, LoginHistoryItem{
			ID:            attempt.ID,
			Timestamp:     attempt.AttemptAt,
			Status:        attempt.Status,
			FailureReason: attempt.FailureReason,
			IPAddress:     attempt.IPAddress,
			DeviceName:    attempt.DeviceName,
			DeviceOS:      attempt.DeviceOS,
		})
	}
	return result, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}

	user, err := s.users.GetByEmail(ctx, normalized)
	if err != nil {
		// Do not leak whether user exists.
		return nil
	}

	rawToken := randomHex(32)
	tokenHash := hashToken(rawToken)
	now := s.nowFn()
	if err := s.recovery.CreatePasswordResetToken(ctx, user.UserID, tokenHash, now, now.Add(time.Hour)); err != nil {
		return err
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req PasswordResetRequest) error {
	if strings.TrimSpace(req.Token) == "" {
		return fmt.Errorf("%w: token is required", domain.ErrInvalidInput)
	}
	if err := domain.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	userID, err := s.recovery.ConsumePasswordResetToken(ctx, hashToken(req.Token), s.nowFn())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrUnauthorized
		}
		return err
	}

	passwordHash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	return s.credentials.UpdatePassword(ctx, userID, passwordHash, s.nowFn())
}

func (s *Service) RequestEmailVerification(ctx context.Context, jwtToken string) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}

	now := s.nowFn()
	token := randomHex(32)
	tokenHash := hashToken(token)
	if err := s.recovery.CreateEmailVerificationToken(ctx, claims.UserID, tokenHash, now, now.Add(24*time.Hour)); err != nil {
		return err
	}
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("%w: token is required", domain.ErrInvalidInput)
	}
	userID, err := s.recovery.ConsumeEmailVerificationToken(ctx, hashToken(token), s.nowFn())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrUnauthorized
		}
		return err
	}
	return s.credentials.SetEmailVerified(ctx, userID, true, s.nowFn())
}

func (s *Service) Setup2FA(ctx context.Context, jwtToken string, req TwoFASetupRequest) (TwoFASetupResponse, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return TwoFASetupResponse{}, domain.ErrUnauthorized
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if action == "" || method == "" {
		return TwoFASetupResponse{}, fmt.Errorf("%w: action and method are required", domain.ErrInvalidInput)
	}
	if method != "sms" && method != "email" && method != "authenticator_app" && method != "totp" {
		return TwoFASetupResponse{}, fmt.Errorf("%w: unsupported method", domain.ErrInvalidInput)
	}
	if method == "totp" {
		method = "authenticator_app"
	}

	now := s.nowFn()
	switch action {
	case "enable":
		isPrimary := false
		enabledMethods, _ := s.mfa.ListEnabledMethods(ctx, claims.UserID)
		if len(enabledMethods) == 0 {
			isPrimary = true
		}
		if err := s.mfa.SetMethodEnabled(ctx, claims.UserID, method, true, isPrimary, now); err != nil {
			return TwoFASetupResponse{}, err
		}

		resp := TwoFASetupResponse{
			Method:  method,
			Enabled: true,
		}
		if method == "authenticator_app" {
			secret := randomBase32(20)
			if err := s.mfa.UpsertTOTPSecret(ctx, claims.UserID, []byte(secret), now); err != nil {
				return TwoFASetupResponse{}, err
			}
			resp.Secret = secret

			backupCodes := make([]string, 0, 10)
			backupHashes := make([]string, 0, 10)
			for i := 0; i < 10; i++ {
				code := strings.ToUpper(randomBase32(5))
				backupCodes = append(backupCodes, code)
				backupHashes = append(backupHashes, hashToken(code))
			}
			if err := s.mfa.ReplaceBackupCodes(ctx, claims.UserID, backupHashes, now); err != nil {
				return TwoFASetupResponse{}, err
			}
			resp.BackupCodes = backupCodes
		}
		return resp, nil

	case "disable":
		enabledMethods, err := s.mfa.ListEnabledMethods(ctx, claims.UserID)
		if err != nil {
			return TwoFASetupResponse{}, err
		}
		if len(enabledMethods) <= 1 {
			return TwoFASetupResponse{}, domain.ErrCannotUnlinkLastAuth
		}
		if err := s.mfa.SetMethodEnabled(ctx, claims.UserID, method, false, false, now); err != nil {
			return TwoFASetupResponse{}, err
		}
		return TwoFASetupResponse{
			Method:  method,
			Enabled: false,
		}, nil

	default:
		return TwoFASetupResponse{}, fmt.Errorf("%w: unsupported action", domain.ErrInvalidInput)
	}
}

func (s *Service) Verify2FA(ctx context.Context, req TwoFAVerifyRequest) (LoginResponse, error) {
	tempToken := strings.TrimSpace(req.TempToken)
	if tempToken == "" {
		return LoginResponse{}, fmt.Errorf("%w: temp token is required", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(req.Code) == "" {
		return LoginResponse{}, fmt.Errorf("%w: code is required", domain.ErrInvalidInput)
	}

	challenge, err := s.challenges.Get(ctx, tempToken)
	if err != nil {
		return LoginResponse{}, err
	}
	if challenge == nil {
		return LoginResponse{}, domain.ErrUnauthorized
	}
	if challenge.ExpiresAt.Before(s.nowFn()) {
		_ = s.challenges.Delete(ctx, tempToken)
		return LoginResponse{}, domain.ErrTokenExpired
	}

	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method != "" && method != strings.ToLower(challenge.Method) {
		return LoginResponse{}, domain.ErrInvalidInput
	}

	valid := req.Code == challenge.Code
	if !valid {
		backupOK, backupErr := s.mfa.ConsumeBackupCode(ctx, challenge.UserID, hashToken(strings.ToUpper(strings.TrimSpace(req.Code))), s.nowFn())
		if backupErr != nil {
			return LoginResponse{}, backupErr
		}
		valid = backupOK
	}
	if !valid {
		return LoginResponse{}, domain.ErrInvalidCredentials
	}

	now := s.nowFn()
	session, err := s.sessions.Create(ctx, ports.SessionCreateParams{
		UserID:         challenge.UserID,
		DeviceName:     req.DeviceName,
		DeviceOS:       req.DeviceOS,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		ExpiresAt:      now.Add(s.cfg.SessionTTL),
		LastActivityAt: now,
	})
	if err != nil {
		return LoginResponse{}, err
	}
	token, err := s.tokenSigner.Sign(ports.AuthClaims{
		UserID:    challenge.UserID,
		Email:     challenge.Email,
		Role:      challenge.Role,
		SessionID: session.SessionID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.cfg.TokenTTL),
	})
	if err != nil {
		return LoginResponse{}, err
	}
	_ = s.challenges.Delete(ctx, tempToken)

	return LoginResponse{
		Token:     token,
		SessionID: session.SessionID,
		ExpiresIn: int64(s.cfg.TokenTTL.Seconds()),
	}, nil
}

func (s *Service) OIDCAuthorize(ctx context.Context, provider, redirectURI, state, nonce, loginHint string) (string, error) {
	if s.oidcVerifier == nil {
		return "", fmt.Errorf("%w: OIDC verifier is not configured", domain.ErrNotImplemented)
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "google"
	}
	if strings.TrimSpace(redirectURI) == "" || strings.TrimSpace(state) == "" || strings.TrimSpace(nonce) == "" {
		return "", fmt.Errorf("%w: redirect_uri, state and nonce are required", domain.ErrInvalidInput)
	}
	if _, err := url.ParseRequestURI(redirectURI); err != nil {
		return "", fmt.Errorf("%w: invalid redirect_uri", domain.ErrInvalidInput)
	}

	codeVerifier, codeChallenge := generatePKCEVerifierChallenge()
	now := s.nowFn()
	if err := s.oidcState.Put(ctx, state, ports.OIDCAuthState{
		Provider:     provider,
		RedirectURI:  redirectURI,
		Nonce:        nonce,
		LoginHint:    strings.ToLower(strings.TrimSpace(loginHint)),
		CodeVerifier: codeVerifier,
		CreatedAt:    now,
		ExpiresAt:    now.Add(10 * time.Minute),
	}, 10*time.Minute); err != nil {
		return "", err
	}

	redirectURL, err := s.oidcVerifier.BuildAuthorizeURL(
		ctx,
		provider,
		redirectURI,
		state,
		nonce,
		strings.ToLower(strings.TrimSpace(loginHint)),
		codeChallenge,
	)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	return redirectURL, nil
}

func (s *Service) OIDCCallback(ctx context.Context, code, state string) (string, error) {
	if s.oidcVerifier == nil {
		return "", fmt.Errorf("%w: OIDC verifier is not configured", domain.ErrNotImplemented)
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		return "", fmt.Errorf("%w: code and state are required", domain.ErrInvalidInput)
	}

	authState, err := s.oidcState.Get(ctx, state)
	if err != nil {
		return "", err
	}
	if authState == nil || authState.ExpiresAt.Before(s.nowFn()) {
		return "", domain.ErrUnauthorized
	}

	identity, err := s.oidcVerifier.ExchangeCode(
		ctx,
		authState.Provider,
		code,
		authState.RedirectURI,
		authState.Nonce,
		authState.CodeVerifier,
	)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	if identity.ProviderSub == "" {
		return "", domain.ErrUnauthorized
	}
	if !identity.EmailVerified {
		return "", domain.ErrUnauthorized
	}

	providerSub := strings.TrimSpace(identity.ProviderSub)
	userID, err := s.oidc.FindUserByProviderSubject(ctx, authState.Provider, providerSub)
	var user domain.User
	if err == nil {
		user, err = s.users.GetByID(ctx, userID)
		if err != nil {
			return "", err
		}
	} else {
		email := strings.ToLower(strings.TrimSpace(identity.Email))
		if email == "" {
			email = authState.LoginHint
		}
		if email == "" {
			email = "oidc-" + providerSub[:12] + "@example.invalid"
		}
		existing, getErr := s.users.GetByEmail(ctx, email)
		if getErr == nil {
			user = existing
		} else {
			created, createErr := s.users.CreateWithOutboxTx(ctx, ports.CreateUserTxParams{
				Email:           email,
				PasswordHash:    "",
				RoleName:        s.cfg.DefaultRole,
				EmailVerified:   identity.EmailVerified,
				IdempotencyKey:  "",
				RegisteredAtUTC: s.nowFn(),
			}, ports.OutboxEvent{
				EventID:      uuid.New(),
				EventType:    "user.registered",
				PartitionKey: email,
				Payload:      []byte(`{}`),
				OccurredAt:   s.nowFn(),
			})
			if createErr != nil {
				return "", createErr
			}
			user = created
		}
	}

	if err := s.oidc.UpsertConnection(ctx, user.UserID, authState.Provider, providerSub, user.Email, true, s.nowFn()); err != nil {
		return "", err
	}

	now := s.nowFn()
	session, err := s.sessions.Create(ctx, ports.SessionCreateParams{
		UserID:         user.UserID,
		DeviceName:     "oidc",
		DeviceOS:       "browser",
		IPAddress:      "",
		UserAgent:      "oidc-callback",
		ExpiresAt:      now.Add(s.cfg.SessionTTL),
		LastActivityAt: now,
	})
	if err != nil {
		return "", err
	}
	token, err := s.tokenSigner.Sign(ports.AuthClaims{
		UserID:    user.UserID,
		Email:     user.Email,
		Role:      user.RoleName,
		SessionID: session.SessionID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.cfg.TokenTTL),
	})
	if err != nil {
		return "", err
	}
	_ = s.oidcState.Delete(ctx, state)

	fragment := "token=" + url.QueryEscape(token) + "&session_id=" + session.SessionID.String()
	return buildRedirectWithFragment(authState.RedirectURI, fragment), nil
}

func (s *Service) LinkOIDC(ctx context.Context, jwtToken string, req OIDCLinkRequest) error {
	if s.oidcVerifier == nil {
		return fmt.Errorf("%w: OIDC verifier is not configured", domain.ErrNotImplemented)
	}
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "google"
	}
	if strings.TrimSpace(req.AuthorizationCode) == "" {
		return fmt.Errorf("%w: authorization_code is required", domain.ErrInvalidInput)
	}
	identity, err := s.oidcVerifier.ExchangeCode(
		ctx,
		provider,
		req.AuthorizationCode,
		strings.TrimSpace(req.RedirectURI),
		strings.TrimSpace(req.Nonce),
		strings.TrimSpace(req.CodeVerifier),
	)
	if err != nil {
		return domain.ErrUnauthorized
	}
	if identity.ProviderSub == "" {
		return domain.ErrUnauthorized
	}
	if !identity.EmailVerified {
		return domain.ErrUnauthorized
	}
	email := claims.Email
	if strings.TrimSpace(identity.Email) != "" {
		email = strings.ToLower(strings.TrimSpace(identity.Email))
	}
	return s.oidc.UpsertConnection(ctx, claims.UserID, provider, identity.ProviderSub, email, false, s.nowFn())
}

func (s *Service) UnlinkOIDC(ctx context.Context, jwtToken, provider string) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return fmt.Errorf("%w: provider is required", domain.ErrInvalidInput)
	}

	hasPassword, err := s.credentials.HasPassword(ctx, claims.UserID)
	if err != nil {
		return err
	}
	count, err := s.oidc.CountConnections(ctx, claims.UserID)
	if err != nil {
		return err
	}
	if !hasPassword && count <= 1 {
		return domain.ErrCannotUnlinkLastAuth
	}

	ok, err := s.oidc.DeleteConnection(ctx, claims.UserID, provider)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Service) ValidateToken(ctx context.Context, token string) (ports.AuthClaims, error) {
	claims, err := s.tokenSigner.ParseAndValidate(token)
	if err != nil {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	if revoked, _ := s.revocations.IsRevoked(ctx, claims.SessionID); revoked {
		return ports.AuthClaims{}, domain.ErrSessionRevoked
	}
	session, err := s.sessions.GetByID(ctx, claims.SessionID)
	if err != nil {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	if session.UserID != claims.UserID {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	if session.RevokedAt != nil {
		return ports.AuthClaims{}, domain.ErrSessionRevoked
	}
	if session.ExpiresAt.Before(s.nowFn()) {
		return ports.AuthClaims{}, domain.ErrSessionExpired
	}
	if session.CreatedAt.Add(s.cfg.SessionAbsoluteTTL).Before(s.nowFn()) {
		return ports.AuthClaims{}, domain.ErrSessionExpired
	}
	return claims, nil
}

func (s *Service) PublicJWKs() ([]map[string]any, error) {
	return s.tokenSigner.PublicJWKs()
}

func (s *Service) recordFailure(ctx context.Context, userID *uuid.UUID, req LoginRequest, reason string) {
	_ = s.loginAttempts.Insert(ctx, domain.LoginAttempt{
		UserID:        userID,
		AttemptAt:     s.nowFn(),
		IPAddress:     req.IPAddress,
		Status:        "FAILED",
		FailureReason: reason,
		DeviceName:    req.DeviceName,
		DeviceOS:      req.DeviceOS,
		UserAgent:     req.UserAgent,
	})
}

func normalizeEmail(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" {
		return "", fmt.Errorf("%w: email is required", domain.ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(trimmed); err != nil {
		return "", fmt.Errorf("%w: invalid email", domain.ErrInvalidInput)
	}
	return trimmed, nil
}

func hashRequest(req any) string {
	raw, _ := json.Marshal(req)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func randomHex(bytesLen int) string {
	raw := make([]byte, bytesLen)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

func randomBase32(bytesLen int) string {
	raw := make([]byte, bytesLen)
	_, _ = rand.Read(raw)
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "=")
}

func randomDigits(size int) string {
	if size <= 0 {
		size = 6
	}
	max := 1
	for i := 0; i < size; i++ {
		max *= 10
	}
	nRaw := make([]byte, 8)
	_, _ = rand.Read(nRaw)
	n := int(nRaw[0])<<24 | int(nRaw[1])<<16 | int(nRaw[2])<<8 | int(nRaw[3])
	if n < 0 {
		n = -n
	}
	value := n % max
	return fmt.Sprintf("%0*d", size, value)
}

func generatePKCEVerifierChallenge() (string, string) {
	verifier := randomBase32(32)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func buildRedirectWithFragment(redirectURI, fragment string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path = path.Clean(u.Path)
	}
	u.Fragment = fragment
	return u.String()
}
