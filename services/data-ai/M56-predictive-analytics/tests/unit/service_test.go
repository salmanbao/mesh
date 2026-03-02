package unit

import (
	"context"
	"testing"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/domain"
)

func newService() *application.Service {
	repos := postgres.NewRepositories()
	return application.NewService(application.Dependencies{
		Config: application.Config{
			ServiceName:    "M56-Predictive-Analytics",
			IdempotencyTTL: 7 * 24 * time.Hour,
			ModelVersion:   "v1.0.0",
		},
		Idempotency: repos.Idempotency,
		Predictions: repos.Predictions,
	})
}

func TestPredictCampaignSuccessIdempotency(t *testing.T) {
	t.Parallel()
	svc := newService()
	actor := application.Actor{
		SubjectID:      "creator-1",
		Role:           "creator",
		RequestID:      "req-1",
		IdempotencyKey: "idem-campaign-1",
	}
	input := application.CampaignSuccessInput{
		CampaignID: "camp-1",
		RewardRate: 1.2,
		Budget:     2500,
		Niche:      "gaming",
	}
	first, err := svc.PredictCampaignSuccess(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first prediction failed: %v", err)
	}
	second, err := svc.PredictCampaignSuccess(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("second prediction failed: %v", err)
	}
	if first.PredictionID != second.PredictionID {
		t.Fatalf("expected idempotent replay to return same prediction")
	}
}

func TestPredictCampaignSuccessRejectsKeyReuseWithDifferentPayload(t *testing.T) {
	t.Parallel()
	svc := newService()
	actor := application.Actor{
		SubjectID:      "creator-1",
		Role:           "creator",
		RequestID:      "req-2",
		IdempotencyKey: "idem-campaign-2",
	}
	_, err := svc.PredictCampaignSuccess(context.Background(), actor, application.CampaignSuccessInput{
		CampaignID: "camp-2",
		RewardRate: 1.0,
		Budget:     1000,
		Niche:      "tech",
	})
	if err != nil {
		t.Fatalf("initial prediction failed: %v", err)
	}
	_, err = svc.PredictCampaignSuccess(context.Background(), actor, application.CampaignSuccessInput{
		CampaignID: "camp-2",
		RewardRate: 2.5,
		Budget:     1000,
		Niche:      "tech",
	})
	if err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestReadPredictionsRequireAuth(t *testing.T) {
	t.Parallel()
	svc := newService()
	_, err := svc.GetViewForecast(context.Background(), application.Actor{}, application.ViewForecastInput{UserID: "u-1", WindowDays: 30})
	if err != domain.ErrUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
