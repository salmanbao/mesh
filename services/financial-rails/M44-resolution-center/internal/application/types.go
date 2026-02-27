package application

import (
	"time"

	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	OutboxFlushBatchSize int
	DefaultCurrency      string
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateDisputeInput struct {
	DisputeType       string
	TransactionID     string
	ReasonCategory    string
	JustificationText string
	RequestedAmount   float64
	EvidenceFiles     []domain.EvidenceFile
}

type SendMessageInput struct {
	MessageBody string
	Attachments []domain.EvidenceFile
}

type ApproveDisputeInput struct {
	RefundAmount    float64
	ApprovalReason  string
	ResolutionNotes string
}

type Service struct {
	cfg Config

	disputes     ports.DisputeRepository
	messages     ports.MessageRepository
	evidence     ports.EvidenceRepository
	approvals    ports.ApprovalRepository
	auditLogs    ports.AuditLogRepository
	stateHistory ports.StateHistoryRepository
	mediation    ports.MediationRepository
	rules        ports.AutoResolutionRuleRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	outbox       ports.OutboxRepository

	moderation ports.ModerationReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config Config

	Disputes     ports.DisputeRepository
	Messages     ports.MessageRepository
	Evidence     ports.EvidenceRepository
	Approvals    ports.ApprovalRepository
	AuditLogs    ports.AuditLogRepository
	StateHistory ports.StateHistoryRepository
	Mediation    ports.MediationRepository
	Rules        ports.AutoResolutionRuleRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository

	Moderation ports.ModerationReader

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M44-Resolution-Center"
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
	if cfg.DefaultCurrency == "" {
		cfg.DefaultCurrency = "USD"
	}
	return &Service{
		cfg:          cfg,
		disputes:     deps.Disputes,
		messages:     deps.Messages,
		evidence:     deps.Evidence,
		approvals:    deps.Approvals,
		auditLogs:    deps.AuditLogs,
		stateHistory: deps.StateHistory,
		mediation:    deps.Mediation,
		rules:        deps.Rules,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		moderation:   deps.Moderation,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		nowFn:        time.Now().UTC,
	}
}

func normalizeRole(raw string) string {
	if role := domain.NormalizeRole(raw); role != "" {
		return role
	}
	return "user"
}

func isStaffRole(role string) bool {
	switch normalizeRole(role) {
	case "agent", "manager", "director", "legal", "admin":
		return true
	default:
		return false
	}
}
