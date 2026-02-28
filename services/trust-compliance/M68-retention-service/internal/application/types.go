package application

import (
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/ports"
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

type CreatePolicyInput struct {
	DataType            string
	RetentionYears      int
	SoftDeleteGraceDays int
	SelectiveRules      map[string][]string
}

type CreatePreviewInput struct {
	PolicyID string
	DataType string
}

type CreateLegalHoldInput struct {
	EntityID  string
	DataType  string
	Reason    string
	ExpiresAt *time.Time
}

type CreateRestorationInput struct {
	EntityID        string
	DataType        string
	Reason          string
	ArchiveLocation string
}

type Service struct {
	cfg          Config
	policies     ports.RetentionPolicyRepository
	previews     ports.DeletionPreviewRepository
	holds        ports.LegalHoldRepository
	restorations ports.RestorationRepository
	deletions    ports.ScheduledDeletionRepository
	audit        ports.AuditRepository
	idempotency  ports.IdempotencyRepository
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Policies     ports.RetentionPolicyRepository
	Previews     ports.DeletionPreviewRepository
	Holds        ports.LegalHoldRepository
	Restorations ports.RestorationRepository
	Deletions    ports.ScheduledDeletionRepository
	Audit        ports.AuditRepository
	Idempotency  ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M68-Retention-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:          cfg,
		policies:     deps.Policies,
		previews:     deps.Previews,
		holds:        deps.Holds,
		restorations: deps.Restorations,
		deletions:    deps.Deletions,
		audit:        deps.Audit,
		idempotency:  deps.Idempotency,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}
