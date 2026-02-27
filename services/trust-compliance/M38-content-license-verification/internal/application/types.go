package application

import (
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/ports"
)

type Config struct {
	ServiceName    string
	IdempotencyTTL time.Duration
	HoldThreshold  float64
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ScanLicenseInput struct {
	SubmissionID      string
	CreatorID         string
	MediaType         string
	MediaURL          string
	DeclaredLicenseID string
}

type FileAppealInput struct {
	SubmissionID       string
	HoldID             string
	CreatorID          string
	CreatorExplanation string
}

type DMCATakedownInput struct {
	SubmissionID     string
	RightsHolderName string
	ContactEmail     string
	Reference        string
}

type Service struct {
	cfg         Config
	matches     ports.MatchRepository
	holds       ports.HoldRepository
	appeals     ports.AppealRepository
	takedowns   ports.DMCATakedownRepository
	audit       ports.AuditRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Matches     ports.MatchRepository
	Holds       ports.HoldRepository
	Appeals     ports.AppealRepository
	Takedowns   ports.DMCATakedownRepository
	Audit       ports.AuditRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M38-Content-License-Verification"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.HoldThreshold <= 0 || cfg.HoldThreshold > 1 {
		cfg.HoldThreshold = 0.95
	}
	return &Service{
		cfg:         cfg,
		matches:     deps.Matches,
		holds:       deps.Holds,
		appeals:     deps.Appeals,
		takedowns:   deps.Takedowns,
		audit:       deps.Audit,
		idempotency: deps.Idempotency,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
