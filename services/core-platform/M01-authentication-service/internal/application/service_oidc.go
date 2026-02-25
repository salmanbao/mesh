package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// OIDCAuthorize creates OIDC authorize URL and stores server-generated state/PKCE metadata.
func (s *Service) OIDCAuthorize(ctx context.Context, provider, redirectURI, clientContext, loginHint, ipAddress string) (OIDCAuthorizeResponse, error) {
	if s.oidcVerifier == nil {
		return OIDCAuthorizeResponse{}, fmt.Errorf("%w: OIDC verifier is not configured", domain.ErrNotImplemented)
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "google"
	}
	identifier := strings.ToLower(strings.TrimSpace(loginHint))
	if identifier == "" {
		identifier = provider
	}
	if ip := strings.TrimSpace(ipAddress); ip != "" {
		if err := s.enforceRateLimit(
			ctx,
			"oidc-authorize:ip:"+ip,
			s.cfg.OIDCAuthorizeRateLimitIPThreshold,
			s.cfg.OIDCAuthorizeRateLimitWindow,
		); err != nil {
			return OIDCAuthorizeResponse{}, err
		}
	}
	if err := s.enforceRateLimit(
		ctx,
		"oidc-authorize:identifier:"+identifier,
		s.cfg.OIDCAuthorizeRateLimitIdentifierThreshold,
		s.cfg.OIDCAuthorizeRateLimitWindow,
	); err != nil {
		return OIDCAuthorizeResponse{}, err
	}
	if strings.TrimSpace(redirectURI) == "" {
		return OIDCAuthorizeResponse{}, fmt.Errorf("%w: redirect_uri is required", domain.ErrInvalidInput)
	}
	if _, err := url.ParseRequestURI(redirectURI); err != nil {
		return OIDCAuthorizeResponse{}, fmt.Errorf("%w: invalid redirect_uri", domain.ErrInvalidInput)
	}
	if !s.isAllowedOIDCRedirectURI(redirectURI) {
		return OIDCAuthorizeResponse{}, fmt.Errorf("%w: redirect_uri is not allowed", domain.ErrInvalidInput)
	}

	state := uuid.NewString()
	nonce := randomHex(16)
	codeVerifier, codeChallenge := generatePKCEVerifierChallenge()
	now := s.nowFn()
	if err := s.oidcState.Put(ctx, state, ports.OIDCAuthState{
		Provider:      provider,
		RedirectURI:   redirectURI,
		Nonce:         nonce,
		LoginHint:     strings.ToLower(strings.TrimSpace(loginHint)),
		ClientContext: strings.TrimSpace(clientContext),
		CodeVerifier:  codeVerifier,
		CreatedAt:     now,
		ExpiresAt:     now.Add(10 * time.Minute),
	}, 10*time.Minute); err != nil {
		return OIDCAuthorizeResponse{}, err
	}

	authorizeURL, err := s.oidcVerifier.BuildAuthorizeURL(
		ctx,
		provider,
		redirectURI,
		state,
		nonce,
		strings.ToLower(strings.TrimSpace(loginHint)),
		codeChallenge,
	)
	if err != nil {
		return OIDCAuthorizeResponse{}, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	return OIDCAuthorizeResponse{
		AuthorizeURL: authorizeURL,
		State:        state,
	}, nil
}

// OIDCCallback exchanges code, validates identity, resolves user, and returns auth or completion outcome.
func (s *Service) OIDCCallback(ctx context.Context, code, state string) (OIDCCallbackResult, error) {
	if s.oidcVerifier == nil {
		return OIDCCallbackResult{}, fmt.Errorf("%w: OIDC verifier is not configured", domain.ErrNotImplemented)
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		return OIDCCallbackResult{}, fmt.Errorf("%w: code and state are required", domain.ErrInvalidInput)
	}

	authState, err := s.oidcState.Get(ctx, state)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	if authState == nil || authState.ExpiresAt.Before(s.nowFn()) {
		slog.Default().WarnContext(ctx, "suspicious oidc callback state mismatch",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "oidc_callback",
			"outcome", "failure",
			"state", state,
		)
		return OIDCCallbackResult{}, domain.ErrUnauthorized
	}
	_ = s.oidcState.Delete(ctx, state)

	identity, err := s.oidcVerifier.ExchangeCode(
		ctx,
		authState.Provider,
		code,
		authState.RedirectURI,
		authState.Nonce,
		authState.CodeVerifier,
	)
	if err != nil {
		slog.Default().WarnContext(ctx, "oidc token exchange or validation failed",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "oidc_callback",
			"outcome", "failure",
			"provider", authState.Provider,
			"error", err,
		)
		return OIDCCallbackResult{}, domain.ErrUnauthorized
	}

	issuer := strings.TrimSpace(identity.Issuer)
	if issuer == "" {
		issuer = strings.TrimSpace(authState.Provider)
	}
	subject := strings.TrimSpace(identity.Subject)
	if subject == "" {
		subject = strings.TrimSpace(identity.ProviderSub)
	}
	if subject == "" {
		return OIDCCallbackResult{}, domain.ErrUnauthorized
	}

	user, createdWithFallbackEmail, err := s.resolveOIDCUser(ctx, identity, issuer, subject)
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	if !user.IsActive || user.DeletedAt != nil {
		return OIDCCallbackResult{}, domain.ErrUnauthorized
	}

	now := s.nowFn()
	if err := s.oidc.UpsertConnection(ctx, user.UserID, issuer, subject, authState.Provider, user.Email, true, now); err != nil {
		return OIDCCallbackResult{}, err
	}
	slog.Default().InfoContext(ctx, "oidc identity linked",
		"service", "M01-Authentication-Service",
		"module", "application",
		"layer", "application",
		"operation", "oidc_callback",
		"outcome", "success",
		"user_id", user.UserID.String(),
		"issuer", issuer,
		"subject", subject,
		"provider", authState.Provider,
	)
	if err := s.persistOIDCToken(ctx, user.UserID, authState.Provider, identity); err != nil {
		return OIDCCallbackResult{}, err
	}

	if createdWithFallbackEmail && s.regCompletion != nil {
		completionToken := uuid.NewString()
		ttl := s.cfg.OIDCCompletionTokenTTL
		if ttl <= 0 {
			ttl = 10 * time.Minute
		}
		if err := s.regCompletion.Put(ctx, completionToken, ports.RegistrationCompletion{
			UserID:    user.UserID,
			Email:     user.Email,
			Role:      user.RoleName,
			ExpiresAt: now.Add(ttl),
		}, ttl); err != nil {
			return OIDCCallbackResult{}, err
		}
		return OIDCCallbackResult{
			RegistrationIncomplete: true,
			CompletionToken:        completionToken,
			RedirectURL:            buildRedirectWithFragment(authState.RedirectURI, "registration_incomplete=true&completion_token="+url.QueryEscape(completionToken)),
		}, nil
	}

	session, token, err := s.issueSessionToken(ctx, user, "oidc", "browser", "", "oidc-callback")
	if err != nil {
		return OIDCCallbackResult{}, err
	}
	return OIDCCallbackResult{
		UserID:    user.UserID,
		Token:     token,
		SessionID: session.SessionID,
		ExpiresIn: int64(s.cfg.TokenTTL.Seconds()),
		RedirectURL: buildRedirectWithFragment(
			authState.RedirectURI,
			"token="+url.QueryEscape(token)+"&session_id="+session.SessionID.String()+"&user_id="+user.UserID.String(),
		),
	}, nil
}

// RegisterComplete finalizes deferred OIDC onboarding and issues a full session.
func (s *Service) RegisterComplete(ctx context.Context, req RegisterCompleteRequest) (RegisterResponse, error) {
	token := strings.TrimSpace(req.CompletionToken)
	if token == "" {
		return RegisterResponse{}, fmt.Errorf("%w: completion_token is required", domain.ErrInvalidInput)
	}
	if s.regCompletion == nil {
		return RegisterResponse{}, domain.ErrNotImplemented
	}

	completion, err := s.regCompletion.Get(ctx, token)
	if err != nil {
		return RegisterResponse{}, err
	}
	if completion == nil || completion.ExpiresAt.Before(s.nowFn()) {
		return RegisterResponse{}, domain.ErrUnauthorized
	}
	_ = s.regCompletion.Delete(ctx, token)

	user, err := s.users.GetByID(ctx, completion.UserID)
	if err != nil {
		return RegisterResponse{}, err
	}
	session, jwtToken, err := s.issueSessionToken(ctx, user, "oidc", "browser", "", "oidc-register-complete")
	if err != nil {
		return RegisterResponse{}, err
	}
	return RegisterResponse{
		UserID:    user.UserID,
		Token:     jwtToken,
		SessionID: session.SessionID,
		ExpiresIn: int64(s.cfg.TokenTTL.Seconds()),
	}, nil
}

// LinkOIDC attaches an external identity to an existing authenticated account.
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

	issuer := strings.TrimSpace(identity.Issuer)
	if issuer == "" {
		issuer = provider
	}
	subject := strings.TrimSpace(identity.Subject)
	if subject == "" {
		subject = strings.TrimSpace(identity.ProviderSub)
	}
	if subject == "" {
		return domain.ErrUnauthorized
	}
	email := claims.Email
	if strings.TrimSpace(identity.Email) != "" {
		email = strings.ToLower(strings.TrimSpace(identity.Email))
	}
	if err := s.oidc.UpsertConnection(ctx, claims.UserID, issuer, subject, provider, email, false, s.nowFn()); err != nil {
		return err
	}
	return s.persistOIDCToken(ctx, claims.UserID, provider, identity)
}

// UnlinkOIDC removes a provider connection while preserving at least one auth path.
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

func (s *Service) resolveOIDCUser(ctx context.Context, identity ports.OIDCIdentity, issuer, subject string) (domain.User, bool, error) {
	userID, err := s.oidc.FindUserByIssuerSubject(ctx, issuer, subject)
	if err == nil {
		user, getErr := s.users.GetByID(ctx, userID)
		if getErr == nil {
			slog.Default().InfoContext(ctx, "oidc identity matched existing mapping",
				"service", "M01-Authentication-Service",
				"module", "application",
				"layer", "application",
				"operation", "resolve_oidc_user",
				"outcome", "success",
				"user_id", userID.String(),
				"issuer", issuer,
				"subject", subject,
			)
		}
		return user, false, getErr
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, false, err
	}

	emailClaim := strings.ToLower(strings.TrimSpace(identity.Email))
	allowEmailLink := s.cfg.OIDCAllowEmailLinking && identity.EmailVerified && emailClaim != ""
	if allowEmailLink {
		existing, getErr := s.users.GetByEmail(ctx, emailClaim)
		if getErr == nil {
			slog.Default().InfoContext(ctx, "oidc identity linked by verified email",
				"service", "M01-Authentication-Service",
				"module", "application",
				"layer", "application",
				"operation", "resolve_oidc_user",
				"outcome", "success",
				"user_id", existing.UserID.String(),
				"email", emailClaim,
				"issuer", issuer,
				"subject", subject,
			)
			return existing, false, nil
		}
		if !errors.Is(getErr, domain.ErrNotFound) {
			return domain.User{}, false, getErr
		}
	}

	createdWithFallbackEmail := false
	email := emailClaim
	if !allowEmailLink {
		email = ""
	}
	if email == "" {
		if len(subject) > 12 {
			email = "oidc-" + subject[:12] + "@example.invalid"
		} else {
			email = "oidc-" + subject + "@example.invalid"
		}
		createdWithFallbackEmail = true
	}

	now := s.nowFn()
	payload := []byte(`{}`)
	created, createErr := s.users.CreateWithOutboxTx(ctx, ports.CreateUserTxParams{
		Email:           email,
		PasswordHash:    "",
		RoleName:        s.cfg.DefaultRole,
		EmailVerified:   identity.EmailVerified,
		IdempotencyKey:  "",
		RegisteredAtUTC: now,
	}, ports.OutboxEvent{
		EventID:      uuid.New(),
		EventType:    eventTypeUserRegistered,
		PartitionKey: email,
		Payload:      payload,
		OccurredAt:   now,
	})
	if createErr == nil {
		slog.Default().InfoContext(ctx, "oidc identity created new local user",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "resolve_oidc_user",
			"outcome", "success",
			"user_id", created.UserID.String(),
			"email", created.Email,
			"issuer", issuer,
			"subject", subject,
		)
		return created, createdWithFallbackEmail, nil
	}
	if errors.Is(createErr, domain.ErrConflict) {
		existing, getErr := s.users.GetByEmail(ctx, email)
		if getErr == nil {
			return existing, false, nil
		}
	}
	return domain.User{}, false, createErr
}

func (s *Service) issueSessionToken(
	ctx context.Context,
	user domain.User,
	deviceName, deviceOS, ipAddress, userAgent string,
) (domain.Session, string, error) {
	now := s.nowFn()
	session, err := s.sessions.Create(ctx, ports.SessionCreateParams{
		UserID:         user.UserID,
		DeviceName:     deviceName,
		DeviceOS:       deviceOS,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		ExpiresAt:      now.Add(s.cfg.SessionTTL),
		LastActivityAt: now,
	})
	if err != nil {
		return domain.Session{}, "", err
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
		return domain.Session{}, "", err
	}
	return session, token, nil
}

func (s *Service) isAllowedOIDCRedirectURI(redirectURI string) bool {
	if len(s.cfg.OIDCAllowedRedirectURIs) == 0 {
		return true
	}
	target := strings.TrimSpace(redirectURI)
	for _, candidate := range s.cfg.OIDCAllowedRedirectURIs {
		if strings.TrimSpace(candidate) == target {
			return true
		}
	}
	return false
}
