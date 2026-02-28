package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/ports"
)

type Repositories struct {
	Developers  *DeveloperRepository
	Sessions    *SessionRepository
	APIKeys     *APIKeyRepository
	Rotations   *APIKeyRotationRepository
	Webhooks    *WebhookRepository
	Deliveries  *WebhookDeliveryRepository
	Usage       *UsageRepository
	Audit       *AuditRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Developers:  &DeveloperRepository{rowsByID: map[string]domain.Developer{}},
		Sessions:    &SessionRepository{rowsByID: map[string]domain.DeveloperSession{}},
		APIKeys:     &APIKeyRepository{rowsByID: map[string]domain.APIKey{}},
		Rotations:   &APIKeyRotationRepository{rowsByID: map[string]domain.APIKeyRotation{}},
		Webhooks:    &WebhookRepository{rowsByID: map[string]domain.Webhook{}},
		Deliveries:  &WebhookDeliveryRepository{rowsByID: map[string]domain.WebhookDelivery{}},
		Usage:       &UsageRepository{rowsByDeveloperID: map[string]domain.DeveloperUsage{}},
		Audit:       &AuditRepository{rows: make([]domain.AuditLog, 0, 128)},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type DeveloperRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.Developer
}

func (r *DeveloperRepository) Create(_ context.Context, row domain.Developer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.DeveloperID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.DeveloperID] = row
	return nil
}

func (r *DeveloperRepository) GetByID(_ context.Context, developerID string) (domain.Developer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[developerID]
	if !ok {
		return domain.Developer{}, domain.ErrNotFound
	}
	return row, nil
}

type SessionRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.DeveloperSession
}

func (r *SessionRepository) Create(_ context.Context, row domain.DeveloperSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.SessionID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.SessionID] = row
	return nil
}

type APIKeyRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.APIKey
}

func (r *APIKeyRepository) Create(_ context.Context, row domain.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.KeyID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.KeyID] = row
	return nil
}

func (r *APIKeyRepository) GetByID(_ context.Context, keyID string) (domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[keyID]
	if !ok {
		return domain.APIKey{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *APIKeyRepository) Update(_ context.Context, row domain.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.KeyID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.KeyID] = row
	return nil
}

type APIKeyRotationRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.APIKeyRotation
}

func (r *APIKeyRotationRepository) Create(_ context.Context, row domain.APIKeyRotation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.RotationID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.RotationID] = row
	return nil
}

type WebhookRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.Webhook
}

func (r *WebhookRepository) Create(_ context.Context, row domain.Webhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.WebhookID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.WebhookID] = row
	return nil
}

func (r *WebhookRepository) GetByID(_ context.Context, webhookID string) (domain.Webhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[webhookID]
	if !ok {
		return domain.Webhook{}, domain.ErrNotFound
	}
	return row, nil
}

type WebhookDeliveryRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.WebhookDelivery
}

func (r *WebhookDeliveryRepository) Create(_ context.Context, row domain.WebhookDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.DeliveryID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.DeliveryID] = row
	return nil
}

type UsageRepository struct {
	mu                sync.Mutex
	rowsByDeveloperID map[string]domain.DeveloperUsage
}

func (r *UsageRepository) CreateOrUpdate(_ context.Context, row domain.DeveloperUsage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rowsByDeveloperID[row.DeveloperID] = row
	return nil
}

func (r *UsageRepository) GetByDeveloperID(_ context.Context, developerID string) (domain.DeveloperUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByDeveloperID[developerID]
	if !ok {
		return domain.DeveloperUsage{}, domain.ErrNotFound
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
