package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/ports"
)

type Repositories struct {
	Integrations *IntegrationRepository
	Credentials  *APICredentialRepository
	Workflows    *WorkflowRepository
	Executions   *WorkflowExecutionRepository
	Webhooks     *WebhookRepository
	Deliveries   *WebhookDeliveryRepository
	Analytics    *AnalyticsRepository
	Logs         *IntegrationLogRepository
	Idempotency  *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Integrations: &IntegrationRepository{rowsByID: map[string]domain.Integration{}},
		Credentials:  &APICredentialRepository{rowsByID: map[string]domain.APICredential{}},
		Workflows:    &WorkflowRepository{rowsByID: map[string]domain.Workflow{}},
		Executions:   &WorkflowExecutionRepository{rowsByID: map[string]domain.WorkflowExecution{}},
		Webhooks:     &WebhookRepository{rowsByID: map[string]domain.Webhook{}},
		Deliveries:   &WebhookDeliveryRepository{rowsByID: map[string]domain.WebhookDelivery{}},
		Analytics:    &AnalyticsRepository{rowsByIntegrationID: map[string]domain.Analytics{}},
		Logs:         &IntegrationLogRepository{rows: make([]domain.IntegrationLog, 0, 256)},
		Idempotency:  &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
	}
}

type IntegrationRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.Integration
}

func (r *IntegrationRepository) Create(_ context.Context, row domain.Integration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.IntegrationID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.IntegrationID] = row
	return nil
}

func (r *IntegrationRepository) GetByID(_ context.Context, integrationID string) (domain.Integration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[integrationID]
	if !ok {
		return domain.Integration{}, domain.ErrNotFound
	}
	return row, nil
}

type APICredentialRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.APICredential
}

func (r *APICredentialRepository) Create(_ context.Context, row domain.APICredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.CredentialID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.CredentialID] = row
	return nil
}

type WorkflowRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.Workflow
}

func (r *WorkflowRepository) Create(_ context.Context, row domain.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.WorkflowID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.WorkflowID] = row
	return nil
}

func (r *WorkflowRepository) GetByID(_ context.Context, workflowID string) (domain.Workflow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rowsByID[workflowID]
	if !ok {
		return domain.Workflow{}, domain.ErrNotFound
	}
	return row, nil
}

func (r *WorkflowRepository) Update(_ context.Context, row domain.Workflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.WorkflowID]; !ok {
		return domain.ErrNotFound
	}
	r.rowsByID[row.WorkflowID] = row
	return nil
}

type WorkflowExecutionRepository struct {
	mu       sync.Mutex
	rowsByID map[string]domain.WorkflowExecution
}

func (r *WorkflowExecutionRepository) Create(_ context.Context, row domain.WorkflowExecution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rowsByID[row.ExecutionID]; ok {
		return domain.ErrConflict
	}
	r.rowsByID[row.ExecutionID] = row
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

type AnalyticsRepository struct {
	mu                  sync.Mutex
	rowsByIntegrationID map[string]domain.Analytics
}

func (r *AnalyticsRepository) CreateOrUpdate(_ context.Context, row domain.Analytics) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rowsByIntegrationID[row.IntegrationID] = row
	return nil
}

type IntegrationLogRepository struct {
	mu   sync.Mutex
	rows []domain.IntegrationLog
}

func (r *IntegrationLogRepository) Append(_ context.Context, row domain.IntegrationLog) error {
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
