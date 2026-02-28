package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/ports"
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

type RegisterDeveloperInput struct {
	Email   string
	AppName string
}

type CreateAPIKeyInput struct {
	DeveloperID string
	Label       string
}

type CreateWebhookInput struct {
	DeveloperID string
	URL         string
	EventType   string
}

type Service struct {
	cfg         Config
	developers  ports.DeveloperRepository
	sessions    ports.SessionRepository
	apiKeys     ports.APIKeyRepository
	rotations   ports.APIKeyRotationRepository
	webhooks    ports.WebhookRepository
	deliveries  ports.WebhookDeliveryRepository
	usage       ports.UsageRepository
	audit       ports.AuditRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Developers  ports.DeveloperRepository
	Sessions    ports.SessionRepository
	APIKeys     ports.APIKeyRepository
	Rotations   ports.APIKeyRotationRepository
	Webhooks    ports.WebhookRepository
	Deliveries  ports.WebhookDeliveryRepository
	Usage       ports.UsageRepository
	Audit       ports.AuditRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M70-Developer-Portal"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:         cfg,
		developers:  deps.Developers,
		sessions:    deps.Sessions,
		apiKeys:     deps.APIKeys,
		rotations:   deps.Rotations,
		webhooks:    deps.Webhooks,
		deliveries:  deps.Deliveries,
		usage:       deps.Usage,
		audit:       deps.Audit,
		idempotency: deps.Idempotency,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
