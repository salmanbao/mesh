package application

import (
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/ports"
)

type Config struct {
	ServiceName    string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type UpdateConsentInput struct {
	UserID      string
	Preferences map[string]bool
	Reason      string
}

type WithdrawConsentInput struct {
	UserID   string
	Category string
	Reason   string
}

type Service struct {
	cfg         Config
	consents    ports.ConsentRepository
	audit       ports.AuditRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Consents    ports.ConsentRepository
	Audit       ports.AuditRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M50-Consent-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:         cfg,
		consents:    deps.Consents,
		audit:       deps.Audit,
		idempotency: deps.Idempotency,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
