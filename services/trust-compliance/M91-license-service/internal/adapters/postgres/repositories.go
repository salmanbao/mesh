package postgres

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/domain"
)

type Repositories struct {
	Licenses    *LicenseRepository
	Activations *ActivationRepository
	Revocations *RevocationRepository
	Configs     *ProductConfigRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	licenses := []domain.License{{
		ID:             "lic-1",
		LicenseKey:     "ABCDE-FGHIJ-KLMNO-PQRST",
		ProductID:      "prod-1",
		TransactionID:  "txn-1",
		UserID:         "user-1",
		Model:          "device_bound",
		MaxActivations: 2,
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
	}}
	configs := map[string]domain.ProductConfig{
		"prod-1": {ID: "cfg-1", ProductID: "prod-1", Model: "device_bound", MaxActivations: 2},
	}
	return &Repositories{
		Licenses:    &LicenseRepository{rows: licenses},
		Activations: &ActivationRepository{rows: map[string][]domain.Activation{}},
		Revocations: &RevocationRepository{rows: map[string][]domain.Revocation{}},
		Configs:     &ProductConfigRepository{rows: configs},
		Idempotency: &IdempotencyRepository{rows: map[string]domain.IdempotencyRecord{}},
	}
}

type LicenseRepository struct {
	mu   sync.Mutex
	rows []domain.License
}

func (r *LicenseRepository) List(_ context.Context) ([]domain.License, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.License, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

func (r *LicenseRepository) GetByKey(_ context.Context, licenseKey string) (domain.License, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.TrimSpace(strings.ToUpper(licenseKey))
	for _, row := range r.rows {
		if strings.ToUpper(row.LicenseKey) == key {
			return row, nil
		}
	}
	return domain.License{}, domain.ErrNotFound
}

func (r *LicenseRepository) Update(_ context.Context, license domain.License) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, row := range r.rows {
		if row.ID == license.ID {
			r.rows[i] = license
			return nil
		}
	}
	return domain.ErrNotFound
}

type ActivationRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.Activation
}

func (r *ActivationRepository) Add(_ context.Context, activation domain.Activation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[activation.LicenseID] = append(r.rows[activation.LicenseID], activation)
	return nil
}

func (r *ActivationRepository) ListByLicense(_ context.Context, licenseID string) ([]domain.Activation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.rows[strings.TrimSpace(licenseID)]
	out := make([]domain.Activation, len(items))
	copy(out, items)
	return out, nil
}

func (r *ActivationRepository) Update(_ context.Context, activation domain.Activation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.rows[activation.LicenseID]
	for i, row := range items {
		if row.ID == activation.ID {
			items[i] = activation
			r.rows[activation.LicenseID] = items
			return nil
		}
	}
	return domain.ErrNotFound
}

type RevocationRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.Revocation
}

func (r *RevocationRepository) Add(_ context.Context, revocation domain.Revocation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[revocation.LicenseID] = append(r.rows[revocation.LicenseID], revocation)
	return nil
}

func (r *RevocationRepository) ListByLicense(_ context.Context, licenseID string) ([]domain.Revocation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.rows[strings.TrimSpace(licenseID)]
	out := make([]domain.Revocation, len(items))
	copy(out, items)
	return out, nil
}

type ProductConfigRepository struct {
	mu   sync.Mutex
	rows map[string]domain.ProductConfig
}

func (r *ProductConfigRepository) GetByProductID(_ context.Context, productID string) (domain.ProductConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg, ok := r.rows[strings.TrimSpace(productID)]
	if !ok {
		return domain.ProductConfig{}, domain.ErrNotFound
	}
	return cfg, nil
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
