package application

import (
	"strings"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	OutboxFlushBatchSize int
	DefaultThreshold     float64
	ModelVersion         string
	PolicyVersion        string
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type ScoreInput struct {
	EventID               string
	EventType             string
	AffiliateID           string
	ReferralToken         string
	ReferrerID            string
	UserID                string
	ConversionID          string
	OrderID               string
	Amount                float64
	ClickIP               string
	UserAgent             string
	DeviceFingerprintHash string
	FormFillTimeMS        int
	MouseMovementCount    int
	KeyboardCPS           float64
	Region                string
	CampaignType          string
	OccurredAt            string
	Metadata              map[string]string
	TraceID               string
	RawPayload            []byte
}

type SubmitDisputeInput struct {
	DecisionID  string
	SubmittedBy string
	EvidenceURL string
}

type Service struct {
	cfg Config

	referralEvents ports.ReferralEventRepository
	decisions      ports.FraudDecisionRepository
	policies       ports.RiskPolicyRepository
	fingerprints   ports.DeviceFingerprintRepository
	clusters       ports.ClusterRepository
	disputes       ports.DisputeCaseRepository
	auditLogs      ports.AuditLogRepository
	idempotency    ports.IdempotencyRepository
	eventDedup     ports.EventDedupRepository
	outbox         ports.OutboxRepository

	affiliate ports.AffiliateReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config Config

	ReferralEvents ports.ReferralEventRepository
	Decisions      ports.FraudDecisionRepository
	Policies       ports.RiskPolicyRepository
	Fingerprints   ports.DeviceFingerprintRepository
	Clusters       ports.ClusterRepository
	Disputes       ports.DisputeCaseRepository
	AuditLogs      ports.AuditLogRepository
	Idempotency    ports.IdempotencyRepository
	EventDedup     ports.EventDedupRepository
	Outbox         ports.OutboxRepository

	Affiliate ports.AffiliateReader

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M96-Referral-Fraud-Detection-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	if cfg.DefaultThreshold <= 0 || cfg.DefaultThreshold > 1 {
		cfg.DefaultThreshold = 0.8
	}
	if cfg.ModelVersion == "" {
		cfg.ModelVersion = "v1.0.0"
	}
	if cfg.PolicyVersion == "" {
		cfg.PolicyVersion = "policy-2026-02-01"
	}
	return &Service{
		cfg:            cfg,
		referralEvents: deps.ReferralEvents,
		decisions:      deps.Decisions,
		policies:       deps.Policies,
		fingerprints:   deps.Fingerprints,
		clusters:       deps.Clusters,
		disputes:       deps.Disputes,
		auditLogs:      deps.AuditLogs,
		idempotency:    deps.Idempotency,
		eventDedup:     deps.EventDedup,
		outbox:         deps.Outbox,
		affiliate:      deps.Affiliate,
		domainEvents:   deps.DomainEvents,
		analytics:      deps.Analytics,
		dlq:            deps.DLQ,
		nowFn:          func() time.Time { return time.Now().UTC() },
	}
}

func normalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin":
		return "admin"
	case "analyst":
		return "analyst"
	default:
		return "user"
	}
}
