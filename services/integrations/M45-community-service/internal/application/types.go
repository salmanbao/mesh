package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ConnectIntegrationInput struct {
	Platform      string
	CommunityName string
	Config        map[string]string
}

type ManualGrantInput struct {
	UserID        string
	ProductID     string
	IntegrationID string
	Reason        string
	Tier          string
}

type Service struct {
	cfg          Config
	integrations ports.CommunityIntegrationRepository
	mappings     ports.ProductCommunityMappingRepository
	grants       ports.CommunityGrantRepository
	auditLogs    ports.CommunityAuditLogRepository
	healthChecks ports.CommunityHealthCheckRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Integrations ports.CommunityIntegrationRepository
	Mappings     ports.ProductCommunityMappingRepository
	Grants       ports.CommunityGrantRepository
	AuditLogs    ports.CommunityAuditLogRepository
	HealthChecks ports.CommunityHealthCheckRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M45-Community-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.ConsumerPollInterval <= 0 {
		cfg.ConsumerPollInterval = 2 * time.Second
	}
	return &Service{cfg: cfg, integrations: deps.Integrations, mappings: deps.Mappings, grants: deps.Grants, auditLogs: deps.AuditLogs, healthChecks: deps.HealthChecks, idempotency: deps.Idempotency, eventDedup: deps.EventDedup, nowFn: func() time.Time { return time.Now().UTC() }}
}
