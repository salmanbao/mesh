package application

import (
	"time"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// Service implements M01 authentication use-cases.
// It depends only on ports to keep domain/application logic independent from infrastructure.
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
	regCompletion ports.RegistrationCompletionStore
	oidcVerifier  ports.OIDCVerifier
	hasher        ports.PasswordHasher
	tokenSigner   ports.TokenSigner
	nowFn         func() time.Time
}

// Dependencies groups all required ports to construct Service.
// Explicit wiring keeps runtime composition transparent and test doubles easy to inject.
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
	RegistrationCompletion ports.RegistrationCompletionStore
	OIDCVerifier  ports.OIDCVerifier
	Hasher        ports.PasswordHasher
	TokenSigner   ports.TokenSigner
}

// NewService builds a fully wired application service.
// The reason for constructor injection is to make boundaries explicit and deterministic in tests.
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
		regCompletion: deps.RegistrationCompletion,
		oidcVerifier:  deps.OIDCVerifier,
		hasher:        deps.Hasher,
		tokenSigner:   deps.TokenSigner,
		nowFn:         time.Now().UTC,
	}
}
