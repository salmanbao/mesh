package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/viralforge/mesh/services/data-ai/M56-predictive-analytics/internal/domain"
)

func (s *Service) GetViewForecast(_ context.Context, actor Actor, input ViewForecastInput) (domain.ViewForecast, error) {
	userID, err := resolveUser(actor, input.UserID)
	if err != nil {
		return domain.ViewForecast{}, err
	}
	window := normalizeWindow(input.WindowDays)
	base := stableScore(userID)
	views := int(float64(window) * (1200 + base*1800))
	low := int(float64(views) * 0.85)
	high := int(float64(views) * 1.15)
	confidence := clamp(0.55+base*0.4, 0, 0.99)

	return domain.ViewForecast{
		UserID:            userID,
		ForecastWindow:    fmt.Sprintf("%dd", window),
		ForecastViews:     views,
		ForecastViewsLow:  low,
		ForecastViewsHigh: high,
		ConfidenceScore:   confidence,
		ModelVersion:      s.cfg.ModelVersion,
		GeneratedAt:       s.nowFn(),
	}, nil
}

func (s *Service) GetClipRecommendations(_ context.Context, actor Actor, input ClipRecommendationsInput) ([]domain.ClipRecommendation, error) {
	userID, err := resolveUser(actor, input.UserID)
	if err != nil {
		return nil, err
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 3
	}
	if limit > 10 {
		limit = 10
	}
	base := stableScore(userID)
	out := make([]domain.ClipRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		score := clamp(base-0.08*float64(i), 0.25, 0.99)
		out = append(out, domain.ClipRecommendation{
			SourceID:      fmt.Sprintf("src_%s_%02d", sanitizeID(userID), i+1),
			Score:         score,
			Reason:        recommendationReason(i),
			ExpectedViews: int(8000 + score*18000),
		})
	}
	return out, nil
}

func (s *Service) GetChurnRisk(_ context.Context, actor Actor, input ChurnRiskInput) (domain.ChurnRisk, error) {
	userID, err := resolveUser(actor, input.UserID)
	if err != nil {
		return domain.ChurnRisk{}, err
	}
	score := clamp(1.0-stableScore(userID), 0.05, 0.95)
	level := "Low"
	action := "Maintain engagement cadence"
	if score >= 0.67 {
		level = "High"
		action = "Trigger retention workflow"
	} else if score >= 0.33 {
		level = "Medium"
		action = "Send nudges and incentive campaign"
	}
	return domain.ChurnRisk{
		UserID:            userID,
		ChurnRiskLevel:    level,
		ChurnRiskScore:    score,
		RecommendedAction: action,
		GeneratedAt:       s.nowFn(),
	}, nil
}

func (s *Service) PredictCampaignSuccess(ctx context.Context, actor Actor, input CampaignSuccessInput) (domain.CampaignSuccessPrediction, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CampaignSuccessPrediction{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CampaignSuccessPrediction{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.CampaignID) == "" || input.RewardRate <= 0 || input.Budget <= 0 {
		return domain.CampaignSuccessPrediction{}, domain.ErrInvalidInput
	}

	requestHash := hashPayload(input)
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.CampaignSuccessPrediction{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.CampaignSuccessPrediction{}, domain.ErrIdempotencyConflict
		}
		var cached domain.CampaignSuccessPrediction
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.CampaignSuccessPrediction{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.CampaignSuccessPrediction{}, err
	}

	rewardComponent := clamp(input.RewardRate/3.0, 0, 1)
	budgetComponent := clamp(math.Log10(input.Budget+1)/4.0, 0, 1)
	nicheComponent := 0.6
	if strings.EqualFold(strings.TrimSpace(input.Niche), "gaming") || strings.EqualFold(strings.TrimSpace(input.Niche), "beauty") {
		nicheComponent = 0.72
	}
	likelihood := clamp(rewardComponent*0.35+budgetComponent*0.40+nicheComponent*0.25, 0.01, 0.99)
	prediction := "Low"
	advice := "Increase reward rate or budget to improve projected success."
	if likelihood >= 0.8 {
		prediction = "High"
		advice = "Campaign is launch-ready with strong projected success."
	} else if likelihood >= 0.6 {
		prediction = "Medium"
		advice = "Small reward-rate increase can push this campaign above 80%."
	}
	row := domain.CampaignSuccessPrediction{
		PredictionID:      fmt.Sprintf("pred_%d", now.UnixNano()),
		CampaignID:        strings.TrimSpace(input.CampaignID),
		SuccessLikelihood: likelihood,
		SuccessPrediction: prediction,
		Advice:            advice,
		ModelVersion:      s.cfg.ModelVersion,
		GeneratedAt:       now,
	}
	if err := s.predictions.SaveCampaignSuccess(ctx, row); err != nil {
		return domain.CampaignSuccessPrediction{}, err
	}
	encoded, err := json.Marshal(row)
	if err != nil {
		return domain.CampaignSuccessPrediction{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 200, encoded, now); err != nil {
		return domain.CampaignSuccessPrediction{}, err
	}
	return row, nil
}

func resolveUser(actor Actor, requested string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return strings.TrimSpace(actor.SubjectID), nil
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if requested != strings.TrimSpace(actor.SubjectID) && role != "admin" && role != "support" {
		return "", domain.ErrForbidden
	}
	return requested, nil
}

func recommendationReason(index int) string {
	reasons := []string{
		"High viral potential from recent trajectory",
		"Strong niche alignment with your history",
		"Similar profile to your top performers",
	}
	if index < len(reasons) {
		return reasons[index]
	}
	return "High expected upside based on model features"
}

func normalizeWindow(window int) int {
	switch window {
	case 7, 30, 90:
		return window
	default:
		return 30
	}
}

func stableScore(seed string) float64 {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(seed))))
	// Use two bytes for deterministic fractional spread.
	value := float64(int(sum[0])<<8|int(sum[1])) / 65535.0
	return clamp(value, 0.1, 0.95)
}

func hashPayload(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func sanitizeID(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, "_", "")
	if raw == "" {
		return "anon"
	}
	if len(raw) > 12 {
		return raw[:12]
	}
	return raw
}
