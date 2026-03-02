package domain

import "time"

type ViewForecast struct {
	UserID            string    `json:"user_id"`
	ForecastWindow    string    `json:"forecast_window"`
	ForecastViews     int       `json:"forecast_views"`
	ForecastViewsLow  int       `json:"forecast_views_low"`
	ForecastViewsHigh int       `json:"forecast_views_high"`
	ConfidenceScore   float64   `json:"confidence_score"`
	ModelVersion      string    `json:"model_version"`
	GeneratedAt       time.Time `json:"generated_at"`
}

type ClipRecommendation struct {
	SourceID      string  `json:"source_id"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
	ExpectedViews int     `json:"expected_views"`
}

type ChurnRisk struct {
	UserID            string    `json:"user_id"`
	ChurnRiskLevel    string    `json:"churn_risk_level"`
	ChurnRiskScore    float64   `json:"churn_risk_score"`
	RecommendedAction string    `json:"recommended_action"`
	GeneratedAt       time.Time `json:"generated_at"`
}

type CampaignSuccessPrediction struct {
	PredictionID      string    `json:"prediction_id"`
	CampaignID        string    `json:"campaign_id"`
	SuccessLikelihood float64   `json:"success_likelihood"`
	SuccessPrediction string    `json:"success_prediction"`
	Advice            string    `json:"advice"`
	ModelVersion      string    `json:"model_version"`
	GeneratedAt       time.Time `json:"generated_at"`
}
