package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// Register creates a local account and emits a registration outbox event in one transaction.
// This guarantees user state and integration signal cannot diverge.
func (s *Service) Register(ctx context.Context, req RegisterRequest, idempotencyKey string) (RegisterResponse, error) {
	if hasDeprecatedOIDCRegisterFields(req) {
		slog.Default().WarnContext(ctx, "deprecated oidc fields received on local register endpoint",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "register",
			"outcome", "warning",
			"metric_name", "auth.register.oidc_fields",
			"metric_value", 1,
			"mode", s.cfg.RegisterOIDCFieldMode,
		)
		if strings.EqualFold(strings.TrimSpace(s.cfg.RegisterOIDCFieldMode), "reject") {
			return RegisterResponse{}, domain.ErrOIDCFlowRequired
		}
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		return RegisterResponse{}, err
	}
	if ip := strings.TrimSpace(req.IPAddress); ip != "" {
		if err := s.enforceRateLimit(
			ctx,
			"register:ip:"+ip,
			s.cfg.RegisterRateLimitIPThreshold,
			s.cfg.RegisterRateLimitWindow,
		); err != nil {
			return RegisterResponse{}, err
		}
	}
	if err := s.enforceRateLimit(
		ctx,
		"register:identifier:"+email,
		s.cfg.RegisterRateLimitIdentifierThreshold,
		s.cfg.RegisterRateLimitWindow,
	); err != nil {
		return RegisterResponse{}, err
	}
	if !req.TermsAccepted {
		return RegisterResponse{}, fmt.Errorf("%w: terms must be accepted", domain.ErrInvalidInput)
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
		EventType:    eventTypeUserRegistered,
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

// Login validates credentials, enforces lockout, and starts MFA or session issuance.
// The split response shape reduces attack surface while preserving UX for MFA flows.
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return LoginResponse{}, err
	}

	lockKey := "login:" + email
	lockState, err := s.lockouts.Get(ctx, lockKey)
	if err == nil && lockState.LockedUntil != nil && lockState.LockedUntil.After(s.nowFn()) {
		slog.Default().WarnContext(ctx, "account lockout active",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "login",
			"outcome", "blocked",
			"email", email,
			"locked_until", lockState.LockedUntil,
		)
		return LoginResponse{}, domain.ErrAccountLocked
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		s.recordFailure(ctx, nil, req, "USER_NOT_FOUND")
		return LoginResponse{}, domain.ErrInvalidCredentials
	}
	if !user.IsActive || user.DeletedAt != nil {
		s.recordFailure(ctx, &user.UserID, req, "ACCOUNT_INACTIVE")
		return LoginResponse{}, domain.ErrInvalidCredentials
	}

	if err := s.hasher.Compare(user.PasswordHash, req.Password); err != nil {
		s.recordFailure(ctx, &user.UserID, req, "INVALID_PASSWORD")
		now := s.nowFn()
		lockState, lockErr := s.lockouts.RecordFailure(ctx, lockKey, now, s.cfg.FailedLoginThreshold, s.cfg.LockoutDuration)
		if lockErr != nil {
			slog.Default().ErrorContext(ctx, "failed to update lockout state",
				"service", "M01-Authentication-Service",
				"module", "application",
				"layer", "application",
				"operation", "login",
				"outcome", "failure",
				"error_code", "LOCKOUT_STATE_UNAVAILABLE",
				"error", lockErr,
			)
			return LoginResponse{}, domain.ErrAccountLocked
		}
		if lockState.LockedUntil != nil && lockState.LockedUntil.After(now) {
			slog.Default().WarnContext(ctx, "account lockout triggered",
				"service", "M01-Authentication-Service",
				"module", "application",
				"layer", "application",
				"operation", "login",
				"outcome", "blocked",
				"email", email,
				"locked_until", lockState.LockedUntil,
			)
			return LoginResponse{}, domain.ErrAccountLocked
		}
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
			EventType:    eventTypeAuth2FARequired,
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

// Refresh rotates an access token for an active, non-revoked session.
// Session-based checks are repeated here to support immediate revocation semantics.
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

// LogoutCurrentSession revokes only the caller's active session.
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

// LogoutAllSessions revokes all sessions for the authenticated user.
// This is used for account hardening after compromise or credential rotation.
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

// DeleteAccount performs user-requested account deletion by deactivating identity and emitting user.deleted.
func (s *Service) DeleteAccount(ctx context.Context, jwtToken string) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return err
	}

	now := s.nowFn()
	if err := s.sessions.RevokeAllByUser(ctx, claims.UserID, now); err != nil {
		return err
	}
	if sessions, listErr := s.sessions.ListByUser(ctx, claims.UserID, 500, 0); listErr == nil {
		for _, session := range sessions {
			_ = s.revocations.MarkRevoked(ctx, session.SessionID, now.Add(s.cfg.TokenTTL))
		}
	}
	if err := s.users.Deactivate(ctx, claims.UserID, now); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]any{
		"user_id":    claims.UserID.String(),
		"email":      user.Email,
		"deleted_at": now,
	})
	if err := s.outbox.Enqueue(ctx, ports.OutboxEvent{
		EventID:      uuid.New(),
		EventType:    eventTypeUserDeleted,
		PartitionKey: claims.UserID.String(),
		Payload:      payload,
		OccurredAt:   now,
	}); err != nil {
		return fmt.Errorf("enqueue user.deleted: %w", err)
	}

	return nil
}

// RevokeSessionByID revokes a specific session owned by the authenticated user.
// Ownership checks prevent cross-user session manipulation.
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

// Verify2FA validates a temporary MFA challenge and issues a full access session.
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

// ValidateToken verifies token integrity and current session validity.
// We re-check session state to support revocation and absolute session expiration.
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

// PublicJWKs returns active public keys for downstream token verification.
func (s *Service) PublicJWKs() ([]map[string]any, error) {
	return s.tokenSigner.PublicJWKs()
}
