package application

import (
	"time"

	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/ports"
)

type Config struct {
	ServiceName    string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type AnalyzeInput struct {
	UserID       string
	ContentID    string
	Content      string
	ModelID      string
	ModelVersion string
}

type BatchItemInput struct {
	ContentID string
	Content   string
}

type BatchAnalyzeInput struct {
	UserID       string
	ModelID      string
	ModelVersion string
	Items        []BatchItemInput
}

type Service struct {
	cfg         Config
	predictions ports.PredictionRepository
	batchJobs   ports.BatchJobRepository
	models      ports.ModelRepository
	feedback    ports.FeedbackRepository
	audit       ports.AuditRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Predictions ports.PredictionRepository
	BatchJobs   ports.BatchJobRepository
	Models      ports.ModelRepository
	Feedback    ports.FeedbackRepository
	Audit       ports.AuditRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M57-AI-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:         cfg,
		predictions: deps.Predictions,
		batchJobs:   deps.BatchJobs,
		models:      deps.Models,
		feedback:    deps.Feedback,
		audit:       deps.Audit,
		idempotency: deps.Idempotency,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
