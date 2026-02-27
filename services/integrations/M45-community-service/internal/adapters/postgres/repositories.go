package postgres

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/ports"
)

type Repositories struct {
	Integrations *CommunityIntegrationRepository
	Mappings     *ProductCommunityMappingRepository
	Grants       *CommunityGrantRepository
	AuditLogs    *CommunityAuditLogRepository
	HealthChecks *CommunityHealthCheckRepository
	Idempotency  *IdempotencyRepository
	EventDedup   *EventDedupRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Integrations: &CommunityIntegrationRepository{rows: map[string]domain.CommunityIntegration{}},
		Mappings:     &ProductCommunityMappingRepository{rows: map[string]domain.ProductCommunityMapping{}},
		Grants:       &CommunityGrantRepository{rows: map[string]domain.CommunityGrant{}},
		AuditLogs:    &CommunityAuditLogRepository{rows: []domain.CommunityAuditLog{}},
		HealthChecks: &CommunityHealthCheckRepository{rows: []domain.CommunityHealthCheck{}},
		Idempotency:  &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:   &EventDedupRepository{rows: map[string]time.Time{}},
	}
}

type CommunityIntegrationRepository struct {
	mu   sync.Mutex
	rows map[string]domain.CommunityIntegration
}

func (r *CommunityIntegrationRepository) Create(_ context.Context, row domain.CommunityIntegration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.IntegrationID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.IntegrationID] = row
	return nil
}
func (r *CommunityIntegrationRepository) GetByID(_ context.Context, integrationID string) (domain.CommunityIntegration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[integrationID]
	if !ok {
		return domain.CommunityIntegration{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *CommunityIntegrationRepository) ListByCreatorID(_ context.Context, creatorID string) ([]domain.CommunityIntegration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.CommunityIntegration{}
	for _, row := range r.rows {
		if row.CreatorID == creatorID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *CommunityIntegrationRepository) Update(_ context.Context, row domain.CommunityIntegration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.IntegrationID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[row.IntegrationID] = row
	return nil
}

type ProductCommunityMappingRepository struct {
	mu   sync.Mutex
	rows map[string]domain.ProductCommunityMapping
}

func (r *ProductCommunityMappingRepository) Create(_ context.Context, row domain.ProductCommunityMapping) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.rows {
		if existing.ProductID == row.ProductID && existing.IntegrationID == row.IntegrationID {
			return domain.ErrConflict
		}
	}
	r.rows[row.MappingID] = row
	return nil
}
func (r *ProductCommunityMappingRepository) FindByProductIntegration(_ context.Context, productID, integrationID string) (domain.ProductCommunityMapping, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.ProductID == productID && row.IntegrationID == integrationID {
			return row, nil
		}
	}
	return domain.ProductCommunityMapping{}, domain.ErrNotFound
}

type CommunityGrantRepository struct {
	mu   sync.Mutex
	rows map[string]domain.CommunityGrant
}

func (r *CommunityGrantRepository) Create(_ context.Context, row domain.CommunityGrant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.GrantID]; ok {
		return domain.ErrConflict
	}
	for _, ex := range r.rows {
		if ex.OrderID == row.OrderID && ex.IntegrationID == row.IntegrationID {
			return domain.ErrConflict
		}
	}
	r.rows[row.GrantID] = row
	return nil
}
func (r *CommunityGrantRepository) GetByID(_ context.Context, grantID string) (domain.CommunityGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[grantID]
	if !ok {
		return domain.CommunityGrant{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *CommunityGrantRepository) FindByOrderIntegration(_ context.Context, orderID, integrationID string) (domain.CommunityGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.OrderID == orderID && row.IntegrationID == integrationID {
			return row, nil
		}
	}
	return domain.CommunityGrant{}, domain.ErrNotFound
}
func (r *CommunityGrantRepository) ListByUserID(_ context.Context, userID string) ([]domain.CommunityGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.CommunityGrant{}
	for _, row := range r.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

type CommunityAuditLogRepository struct {
	mu   sync.Mutex
	rows []domain.CommunityAuditLog
}

func (r *CommunityAuditLogRepository) Append(_ context.Context, row domain.CommunityAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *CommunityAuditLogRepository) List(_ context.Context, userID string, from, to *time.Time) ([]domain.CommunityAuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.CommunityAuditLog{}
	for _, row := range r.rows {
		if userID != "" && row.UserID != userID {
			continue
		}
		if from != nil && row.Timestamp.Before(*from) {
			continue
		}
		if to != nil && row.Timestamp.After(*to) {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out, nil
}

type CommunityHealthCheckRepository struct {
	mu   sync.Mutex
	rows []domain.CommunityHealthCheck
}

func (r *CommunityHealthCheckRepository) Append(_ context.Context, row domain.CommunityHealthCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *CommunityHealthCheckRepository) LatestByIntegrationID(_ context.Context, integrationID string) (domain.CommunityHealthCheck, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *domain.CommunityHealthCheck
	for i := range r.rows {
		if r.rows[i].IntegrationID != integrationID {
			continue
		}
		if latest == nil || r.rows[i].CheckedAt.After(latest.CheckedAt) {
			cp := r.rows[i]
			latest = &cp
		}
	}
	if latest == nil {
		return domain.CommunityHealthCheck{}, domain.ErrNotFound
	}
	return *latest, nil
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

type EventDedupRepository struct {
	mu   sync.Mutex
	rows map[string]time.Time
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	exp, ok := r.rows[eventID]
	if !ok {
		return false, nil
	}
	if now.After(exp) {
		delete(r.rows, eventID)
		return false, nil
	}
	return true, nil
}
func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, _ string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[eventID] = expiresAt
	return nil
}
