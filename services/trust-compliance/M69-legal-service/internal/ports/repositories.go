package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/domain"
)

type DocumentRepository interface {
	Create(ctx context.Context, row domain.LegalDocument) error
	GetByID(ctx context.Context, documentID string) (domain.LegalDocument, error)
}

type SignatureRepository interface {
	Create(ctx context.Context, row domain.SignatureRequest) error
	ListByDocumentID(ctx context.Context, documentID string) ([]domain.SignatureRequest, error)
}

type HoldRepository interface {
	Create(ctx context.Context, row domain.LegalHold) error
	GetByID(ctx context.Context, holdID string) (domain.LegalHold, error)
	Update(ctx context.Context, row domain.LegalHold) error
	GetActiveByEntity(ctx context.Context, entityType, entityID string) (*domain.LegalHold, error)
}

type ComplianceRepository interface {
	CreateReport(ctx context.Context, row domain.ComplianceReport) error
	CreateFinding(ctx context.Context, row domain.ComplianceFinding) error
	GetReportByID(ctx context.Context, reportID string) (domain.ComplianceReport, error)
}

type DisputeRepository interface {
	Create(ctx context.Context, row domain.Dispute) error
	GetByID(ctx context.Context, disputeID string) (domain.Dispute, error)
}

type DMCANoticeRepository interface {
	Create(ctx context.Context, row domain.DMCANotice) error
}

type FilingRepository interface {
	Create(ctx context.Context, row domain.RegulatoryFiling) error
	GetByID(ctx context.Context, filingID string) (domain.RegulatoryFiling, error)
}

type AuditRepository interface {
	Append(ctx context.Context, row domain.AuditLog) error
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}
