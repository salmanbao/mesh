package application

import (
	"time"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/ports"
)

type Config struct {
	ServiceName    string
	IdempotencyTTL time.Duration
	ModelVersion   string
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ViewForecastInput struct {
	UserID     string
	WindowDays int
}

type ClipRecommendationsInput struct {
	UserID string
	Limit  int
}

type ChurnRiskInput struct {
	UserID string
}

type CampaignSuccessInput struct {
	CampaignID string
	RewardRate float64
	Budget     float64
	Niche      string
}

type Service struct {
	cfg         Config
	idempotency ports.IdempotencyRepository
	predictions ports.PredictionRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config Config

	Idempotency ports.IdempotencyRepository
	Predictions ports.PredictionRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M56-Predictive-Analytics"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.ModelVersion == "" {
		cfg.ModelVersion = "v1.0.0"
	}
	return &Service{
		cfg:         cfg,
		idempotency: deps.Idempotency,
		predictions: deps.Predictions,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
