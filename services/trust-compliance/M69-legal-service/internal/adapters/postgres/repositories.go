package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/ports"
)

type Repositories struct {
	Documents   *DocumentRepository
	Signatures  *SignatureRepository
	Holds       *HoldRepository
	Compliance  *ComplianceRepository
	Disputes    *DisputeRepository
	DMCANotices *DMCANoticeRepository
	Filings     *FilingRepository
	Audit       *AuditRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Documents:   &DocumentRepository{rowsByID: map[string]domain.LegalDocument{}},
		Signatures:  &SignatureRepository{rowsByID: map[string]domain.SignatureRequest{}},
		Holds:       &HoldRepository{rowsByID: map[string]domain.LegalHold{}},
		Compliance:  &ComplianceRepository{reportsByID: map[string]domain.ComplianceReport{}, findingsByID: map[string]domain.ComplianceFinding{}},
		Disputes:    &DisputeRepository{rowsByID: map[string]domain.Dispute{}},
		DMCANotices: &DMCANoticeRepository{rowsByID: map[string]domain.DMCANotice{}},
		Filings:     &FilingRepository{rowsByID: map[string]domain.RegulatoryFiling{}},
		Audit:       &AuditRepository{rows: make([]domain.AuditLog, 0, 128)},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type DocumentRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.LegalDocument
}

func (r *DocumentRepository) Create(_ context.Context, row domain.LegalDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.DocumentID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.DocumentID] = row
	return nil
}

func (r *DocumentRepository) GetByID(_ context.Context, documentID string) (domain.LegalDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[documentID]
	if !ok {
		return domain.LegalDocument{}, domain.ErrNotFound
	}
	return row, nil
}

type SignatureRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.SignatureRequest
}

func (r *SignatureRepository) Create(_ context.Context, row domain.SignatureRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.SignatureID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.SignatureID] = row
	return nil
}

func (r *SignatureRepository) ListByDocumentID(_ context.Context, documentID string) ([]domain.SignatureRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.SignatureRequest, 0, len(r.rowsByID))
	for _, row := range r.rowsByID {
		if row.DocumentID == documentID {
			items = append(items, row)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RequestedAt.Before(items[j].RequestedAt) })
	return append([]domain.SignatureRequest(nil), items...), nil
}

type HoldRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.LegalHold
}

func (r *HoldRepository) Create(_ context.Context, row domain.LegalHold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.HoldID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.HoldID] = row
	return nil
}

func (r *HoldRepository) GetByID(_ context.Context, holdID string) (domain.LegalHold, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[holdID]
	if !ok {
		return domain.LegalHold{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *HoldRepository) Update(_ context.Context, row domain.LegalHold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.HoldID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.HoldID] = row
	return nil
}

func (r *HoldRepository) GetActiveByEntity(_ context.Context, entityType, entityID string) (*domain.LegalHold, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rowsByID {
		if row.EntityType == entityType && row.EntityID == entityID && row.Status == domain.HoldStatusActive {
			cp := row
			return &cp, nil
		}
	}
	return nil, nil
}

type ComplianceRepository struct {
	mu           sync.Mutex
	reportsByID  map[string]domain.ComplianceReport
	findingsByID map[string]domain.ComplianceFinding
}

func (r *ComplianceRepository) CreateReport(_ context.Context, row domain.ComplianceReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.reportsByID[row.ReportID]; ok {
		return domain.ErrConflict
	}
	r.reportsByID[row.ReportID] = row
	return nil
}

func (r *ComplianceRepository) CreateFinding(_ context.Context, row domain.ComplianceFinding) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.findingsByID[row.FindingID]; ok {
		return domain.ErrConflict
	}
	r.findingsByID[row.FindingID] = row
	return nil
}

func (r *ComplianceRepository) GetReportByID(_ context.Context, reportID string) (domain.ComplianceReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.reportsByID[reportID]
	if !ok {
		return domain.ComplianceReport{}, domain.ErrNotFound
	}
	return row, nil
}

type DisputeRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.Dispute
}

func (r *DisputeRepository) Create(_ context.Context, row domain.Dispute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.DisputeID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.DisputeID] = row
	return nil
}

func (r *DisputeRepository) GetByID(_ context.Context, disputeID string) (domain.Dispute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[disputeID]
	if !ok {
		return domain.Dispute{}, domain.ErrNotFound
	}
	return row, nil
}

type DMCANoticeRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.DMCANotice
}

func (r *DMCANoticeRepository) Create(_ context.Context, row domain.DMCANotice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.NoticeID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.NoticeID] = row
	return nil
}

type FilingRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.RegulatoryFiling
}

func (r *FilingRepository) Create(_ context.Context, row domain.RegulatoryFiling) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.FilingID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.FilingID] = row
	return nil
}

func (r *FilingRepository) GetByID(_ context.Context, filingID string) (domain.RegulatoryFiling, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[filingID]
	if !ok {
		return domain.RegulatoryFiling{}, domain.ErrNotFound
	}
	return row, nil
}

type AuditRepository struct {
	mu   sync.Mutex
	rows []domain.AuditLog
}

func (r *AuditRepository) Append(_ context.Context, row domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok {
		return nil, nil
	}
	if !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	cp := row
	cp.ResponseBody = append([]byte(nil), row.ResponseBody...)
	return &cp, nil
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[key]; ok {
		if row.RequestHash != requestHash {
			return domain.ErrIdempotencyConflict
		}
		return nil
	}
	r.rows[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}

func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[key]
	row.ResponseCode = responseCode
	row.ResponseBody = append([]byte(nil), responseBody...)
	if row.ExpiresAt.IsZero() {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.rows[key] = row
	return nil
}
