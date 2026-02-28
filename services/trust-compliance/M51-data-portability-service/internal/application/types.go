package application

import (
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/ports"
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

type CreateExportInput struct {
	UserID string
	Format string
}

type EraseInput struct {
	UserID string
	Reason string
}

type Service struct {
	cfg         Config
	exports     ports.ExportRequestRepository
	audit       ports.AuditRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Exports     ports.ExportRequestRepository
	Audit       ports.AuditRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M51-Data-Portability-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:         cfg,
		exports:     deps.Exports,
		audit:       deps.Audit,
		idempotency: deps.Idempotency,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
