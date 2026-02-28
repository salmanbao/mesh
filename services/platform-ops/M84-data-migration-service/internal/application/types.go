package application

import "time"

type Config struct {
	ServiceName    string
	Version        string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
	MFAVerified    bool
}

type CreatePlanInput struct {
	ServiceName string         `json:"service_name"`
	Environment string         `json:"environment"`
	Version     string         `json:"version"`
	Plan        map[string]any `json:"plan"`
	DryRun      bool           `json:"dry_run,omitempty"`
	RiskLevel   string         `json:"risk_level,omitempty"`
}

type CreateRunInput struct {
	PlanID string `json:"plan_id"`
}
