package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/domain"
)

type Repositories struct {
	Configs      *ConfigRepository
	Purges       *PurgeRepository
	Metrics      *MetricsRepository
	Certificates *CertificateRepository
	Idempotency  *IdempotencyRepository
}

func NewRepositories() *Repositories {
	certs := []domain.Certificate{{
		CertID:     "cert-primary",
		Provider:   "cloudflare",
		Domain:     "cdn.example.com",
		ExpiresAt:  time.Now().UTC().Add(90 * 24 * time.Hour),
		AutoRenew:  true,
		TLSVersion: "TLS1.3",
	}}
	return &Repositories{
		Configs:      &ConfigRepository{rows: []domain.CDNConfig{}},
		Purges:       &PurgeRepository{rows: []domain.PurgeRequest{}},
		Metrics:      &MetricsRepository{snapshot: domain.Metrics{HitRate: 0.98, BandwidthGB: 120.5, EgressCostUSD: 42.3, P95LatencyMS: 87, ErrorRate: 0.01, OriginHealthy: true}},
		Certificates: &CertificateRepository{rows: certs},
		Idempotency:  &IdempotencyRepository{rows: map[string]domain.IdempotencyRecord{}},
	}
}

type ConfigRepository struct {
	mu   sync.Mutex
	rows []domain.CDNConfig
}

func (r *ConfigRepository) Create(_ context.Context, config domain.CDNConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows {
		if row.ConfigID == config.ConfigID {
			return domain.ErrConflict
		}
	}
	r.rows = append(r.rows, config)
	sort.Slice(r.rows, func(i, j int) bool { return r.rows[i].Version < r.rows[j].Version })
	return nil
}

func (r *ConfigRepository) List(_ context.Context) ([]domain.CDNConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.CDNConfig, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

func (r *ConfigRepository) Latest(_ context.Context) (domain.CDNConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.rows) == 0 {
		return domain.CDNConfig{}, domain.ErrNotFound
	}
	return r.rows[len(r.rows)-1], nil
}

type PurgeRepository struct {
	mu   sync.Mutex
	rows []domain.PurgeRequest
}

func (r *PurgeRepository) Create(_ context.Context, request domain.PurgeRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, request)
	return nil
}

func (r *PurgeRepository) List(_ context.Context) ([]domain.PurgeRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.PurgeRequest, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

type MetricsRepository struct {
	mu       sync.Mutex
	snapshot domain.Metrics
}

func (r *MetricsRepository) Snapshot(_ context.Context) (domain.Metrics, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot, nil
}

func (r *MetricsRepository) SetSnapshot(_ context.Context, metrics domain.Metrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = metrics
	return nil
}

type CertificateRepository struct {
	mu   sync.Mutex
	rows []domain.Certificate
}

func (r *CertificateRepository) List(_ context.Context) ([]domain.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Certificate, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.rows[strings.TrimSpace(key)]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	copy := rec
	return &copy, nil
}

func (r *IdempotencyRepository) Upsert(_ context.Context, rec domain.IdempotencyRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[rec.Key] = rec
	return nil
}
