package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/domain"
	"github.com/viralforge/mesh/services/trust-compliance/M38-content-license-verification/internal/ports"
)

type Repositories struct {
	Matches     *MatchRepository
	Holds       *HoldRepository
	Appeals     *AppealRepository
	Takedowns   *DMCATakedownRepository
	Audit       *AuditRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Matches:     &MatchRepository{rowsByID: map[string]domain.CopyrightMatch{}, rowBySubmission: map[string]string{}},
		Holds:       &HoldRepository{rowsByID: map[string]domain.LicenseHold{}, rowBySubmission: map[string]string{}},
		Appeals:     &AppealRepository{rowsByID: map[string]domain.LicenseAppeal{}},
		Takedowns:   &DMCATakedownRepository{rowsByID: map[string]domain.DMCATakedown{}},
		Audit:       &AuditRepository{rows: make([]domain.AuditLog, 0, 128)},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type MatchRepository struct {
	mu              sync.Mutex
	rowsByID        map[string]domain.CopyrightMatch
	rowBySubmission map[string]string
}

func (r *MatchRepository) Create(_ context.Context, row domain.CopyrightMatch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.MatchID]; ok {
		return domain.ErrConflict
	}
	if existingID, ok := r.rowBySubmission[row.SubmissionID]; ok && existingID != "" {
		return domain.ErrConflict
	}
	r.rowsByID[row.MatchID] = row
	r.rowBySubmission[row.SubmissionID] = row.MatchID
	return nil
}

func (r *MatchRepository) GetBySubmissionID(_ context.Context, submissionID string) (domain.CopyrightMatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.rowBySubmission[submissionID]
	if !ok {
		return domain.CopyrightMatch{}, domain.ErrNotFound
	}
	row, ok := r.rowsByID[id]
	if !ok {
		return domain.CopyrightMatch{}, domain.ErrNotFound
	}
	return row, nil
}

type HoldRepository struct {
	mu              sync.Mutex
	rowsByID        map[string]domain.LicenseHold
	rowBySubmission map[string]string
}

func (r *HoldRepository) Create(_ context.Context, row domain.LicenseHold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.HoldID]; ok {
		return domain.ErrConflict
	}
	if existingID, ok := r.rowBySubmission[row.SubmissionID]; ok && existingID != "" {
		return domain.ErrConflict
	}
	r.rowsByID[row.HoldID] = row
	r.rowBySubmission[row.SubmissionID] = row.HoldID
	return nil
}

func (r *HoldRepository) GetBySubmissionID(_ context.Context, submissionID string) (domain.LicenseHold, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.rowBySubmission[submissionID]
	if !ok {
		return domain.LicenseHold{}, domain.ErrNotFound
	}
	row, ok := r.rowsByID[id]
	if !ok {
		return domain.LicenseHold{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *HoldRepository) GetByID(_ context.Context, holdID string) (domain.LicenseHold, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[holdID]
	if !ok {
		return domain.LicenseHold{}, domain.ErrNotFound
	}
	return row, nil
}

type AppealRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.LicenseAppeal
}

func (r *AppealRepository) Create(_ context.Context, row domain.LicenseAppeal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.AppealID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.AppealID] = row
	return nil
}

type DMCATakedownRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.DMCATakedown
}

func (r *DMCATakedownRepository) Create(_ context.Context, row domain.DMCATakedown) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.DMCAID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.DMCAID] = row
	return nil
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
