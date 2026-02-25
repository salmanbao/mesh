package postgres

import (
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
)

// Repositories bundles concrete Postgres implementations for all persistence ports.
// Runtime wiring uses this struct to keep adapter construction centralized.
type Repositories struct {
	Users         ports.UserRepository
	Sessions      ports.SessionRepository
	LoginAttempts ports.LoginAttemptRepository
	Outbox        ports.OutboxRepository
	Idempotency   ports.IdempotencyRepository
	Recovery      ports.RecoveryRepository
	Credentials   ports.CredentialRepository
	MFA           ports.MFARepository
	OIDC          ports.OIDCRepository
}

// NewRepositories builds all port implementations backed by a shared GORM handle.
func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{
		Users:         &userRepository{db: db},
		Sessions:      &sessionRepository{db: db},
		LoginAttempts: &loginAttemptRepository{db: db},
		Outbox:        &outboxRepository{db: db},
		Idempotency:   &idempotencyRepository{db: db},
		Recovery:      &recoveryRepository{db: db},
		Credentials:   &credentialRepository{db: db},
		MFA:           &mfaRepository{db: db},
		OIDC:          &oidcRepository{db: db},
	}
}
