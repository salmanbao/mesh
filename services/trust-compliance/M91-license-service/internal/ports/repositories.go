package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M91-license-service/internal/domain"
)

type LicenseRepository interface {
	List(ctx context.Context) ([]domain.License, error)
	GetByKey(ctx context.Context, licenseKey string) (domain.License, error)
	Update(ctx context.Context, license domain.License) error
}

type ActivationRepository interface {
	Add(ctx context.Context, activation domain.Activation) error
	ListByLicense(ctx context.Context, licenseID string) ([]domain.Activation, error)
	Update(ctx context.Context, activation domain.Activation) error
}

type RevocationRepository interface {
	Add(ctx context.Context, revocation domain.Revocation) error
	ListByLicense(ctx context.Context, licenseID string) ([]domain.Revocation, error)
}

type ProductConfigRepository interface {
	GetByProductID(ctx context.Context, productID string) (domain.ProductConfig, error)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error)
	Upsert(ctx context.Context, rec domain.IdempotencyRecord) error
}
