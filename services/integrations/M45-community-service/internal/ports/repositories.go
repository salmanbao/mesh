package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/domain"
)

type CommunityIntegrationRepository interface {
	Create(ctx context.Context, row domain.CommunityIntegration) error
	GetByID(ctx context.Context, integrationID string) (domain.CommunityIntegration, error)
	ListByCreatorID(ctx context.Context, creatorID string) ([]domain.CommunityIntegration, error)
	Update(ctx context.Context, row domain.CommunityIntegration) error
}

type ProductCommunityMappingRepository interface {
	Create(ctx context.Context, row domain.ProductCommunityMapping) error
	FindByProductIntegration(ctx context.Context, productID, integrationID string) (domain.ProductCommunityMapping, error)
}

type CommunityGrantRepository interface {
	Create(ctx context.Context, row domain.CommunityGrant) error
	GetByID(ctx context.Context, grantID string) (domain.CommunityGrant, error)
	FindByOrderIntegration(ctx context.Context, orderID, integrationID string) (domain.CommunityGrant, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.CommunityGrant, error)
}

type CommunityAuditLogRepository interface {
	Append(ctx context.Context, row domain.CommunityAuditLog) error
	List(ctx context.Context, userID string, from, to *time.Time) ([]domain.CommunityAuditLog, error)
}

type CommunityHealthCheckRepository interface {
	Append(ctx context.Context, row domain.CommunityHealthCheck) error
	LatestByIntegrationID(ctx context.Context, integrationID string) (domain.CommunityHealthCheck, error)
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

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}
