package application

import (
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/ports"
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

type UploadDocumentInput struct {
	DocumentType string
	FileName     string
}

type RequestSignatureInput struct {
	SignerUserID string
}

type CreateHoldInput struct {
	EntityType string
	EntityID   string
	Reason     string
}

type ComplianceScanInput struct {
	ReportType string
}

type CreateDisputeInput struct {
	UserID        string
	OpposingParty string
	DisputeReason string
	AmountCents   int64
}

type CreateDMCANoticeInput struct {
	ContentID string
	Claimant  string
	Reason    string
}

type GenerateFilingInput struct {
	UserID  string
	TaxYear int
}

type Service struct {
	cfg         Config
	documents   ports.DocumentRepository
	signatures  ports.SignatureRepository
	holds       ports.HoldRepository
	compliance  ports.ComplianceRepository
	disputes    ports.DisputeRepository
	dmca        ports.DMCANoticeRepository
	filings     ports.FilingRepository
	audit       ports.AuditRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Documents   ports.DocumentRepository
	Signatures  ports.SignatureRepository
	Holds       ports.HoldRepository
	Compliance  ports.ComplianceRepository
	Disputes    ports.DisputeRepository
	DMCA        ports.DMCANoticeRepository
	Filings     ports.FilingRepository
	Audit       ports.AuditRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M69-Legal-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:         cfg,
		documents:   deps.Documents,
		signatures:  deps.Signatures,
		holds:       deps.Holds,
		compliance:  deps.Compliance,
		disputes:    deps.Disputes,
		dmca:        deps.DMCA,
		filings:     deps.Filings,
		audit:       deps.Audit,
		idempotency: deps.Idempotency,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
