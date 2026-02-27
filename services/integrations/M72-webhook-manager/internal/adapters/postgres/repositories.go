package postgres

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/domain"
)

type Repositories struct {
	Webhooks    *WebhookRepository
	Deliveries  *DeliveryRepository
	Analytics   *AnalyticsRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	return &Repositories{
		Webhooks:    &WebhookRepository{rows: map[string]domain.Webhook{}},
		Deliveries:  &DeliveryRepository{rows: map[string][]domain.Delivery{}},
		Analytics:   &AnalyticsRepository{rows: map[string]domain.Analytics{}},
		Idempotency: &IdempotencyRepository{rows: map[string]domain.IdempotencyRecord{}},
	}
}

type WebhookRepository struct {
	mu   sync.Mutex
	rows map[string]domain.Webhook
}

func (r *WebhookRepository) Create(_ context.Context, wh domain.Webhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[wh.WebhookID]; ok {
		return domain.ErrConflict
	}
	r.rows[wh.WebhookID] = wh
	return nil
}

func (r *WebhookRepository) Update(_ context.Context, wh domain.Webhook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[wh.WebhookID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[wh.WebhookID] = wh
	return nil
}

func (r *WebhookRepository) Get(_ context.Context, id string) (domain.Webhook, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	wh, ok := r.rows[strings.TrimSpace(id)]
	if !ok {
		return domain.Webhook{}, domain.ErrNotFound
	}
	return wh, nil
}

type DeliveryRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.Delivery
}

func (r *DeliveryRepository) Add(_ context.Context, d domain.Delivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[d.WebhookID] = append(r.rows[d.WebhookID], d)
	return nil
}

func (r *DeliveryRepository) ListByWebhook(_ context.Context, webhookID string, limit int) ([]domain.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	list := r.rows[strings.TrimSpace(webhookID)]
	if len(list) > limit {
		list = list[len(list)-limit:]
	}
	out := make([]domain.Delivery, len(list))
	copy(out, list)
	return out, nil
}

type AnalyticsRepository struct {
	mu   sync.Mutex
	rows map[string]domain.Analytics
}

func (r *AnalyticsRepository) Snapshot(_ context.Context, webhookID string) (domain.Analytics, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if snap, ok := r.rows[webhookID]; ok {
		return snap, nil
	}
	return domain.Analytics{ByEventType: map[string]domain.Metrics{}}, nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.rows[key]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	cp := rec
	return &cp, nil
}

func (r *IdempotencyRepository) Upsert(_ context.Context, rec domain.IdempotencyRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[rec.Key] = rec
	return nil
}
