package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/domain"
)

type AffiliateRepository interface {
	Create(ctx context.Context, row domain.Affiliate) error
	GetByID(ctx context.Context, affiliateID string) (domain.Affiliate, error)
	GetByUserID(ctx context.Context, userID string) (domain.Affiliate, error)
	Update(ctx context.Context, row domain.Affiliate) error
}

type ReferralLinkRepository interface {
	Create(ctx context.Context, row domain.ReferralLink) error
	GetByID(ctx context.Context, linkID string) (domain.ReferralLink, error)
	GetByToken(ctx context.Context, token string) (domain.ReferralLink, error)
	ListByAffiliateID(ctx context.Context, affiliateID string) ([]domain.ReferralLink, error)
}

type ReferralClickRepository interface {
	Append(ctx context.Context, row domain.ReferralClick) error
	GetByID(ctx context.Context, clickID string) (domain.ReferralClick, error)
	ListByAffiliateID(ctx context.Context, affiliateID string) ([]domain.ReferralClick, error)
	CountByLinkID(ctx context.Context, linkID string) (int, error)
}

type ReferralAttributionRepository interface {
	Create(ctx context.Context, row domain.ReferralAttribution) error
	GetByOrderID(ctx context.Context, orderID string) (domain.ReferralAttribution, error)
	ListByAffiliateID(ctx context.Context, affiliateID string) ([]domain.ReferralAttribution, error)
}

type AffiliateEarningRepository interface {
	Create(ctx context.Context, row domain.AffiliateEarning) error
	ListByAffiliateID(ctx context.Context, affiliateID string) ([]domain.AffiliateEarning, error)
	SumByAffiliateAndStatus(ctx context.Context, affiliateID, status string) (float64, error)
}

type AffiliatePayoutRepository interface {
	Create(ctx context.Context, row domain.AffiliatePayout) error
	ListByAffiliateID(ctx context.Context, affiliateID string) ([]domain.AffiliatePayout, error)
}

type AffiliateAuditLogRepository interface {
	Append(ctx context.Context, row domain.AffiliateAuditLog) error
	ListByAffiliateID(ctx context.Context, affiliateID string) ([]domain.AffiliateAuditLog, error)
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

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
