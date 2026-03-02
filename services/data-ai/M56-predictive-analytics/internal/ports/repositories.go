package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/domain"
)

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

type PredictionRepository interface {
	SaveCampaignSuccess(ctx context.Context, row domain.CampaignSuccessPrediction) error
}
