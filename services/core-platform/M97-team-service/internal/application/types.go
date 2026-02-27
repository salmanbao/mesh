package application

import (
	"strings"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	OutboxFlushBatchSize int
	InviteTTL            time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateTeamInput struct {
	ScopeType string
	ScopeID   string
}

type CreateInviteInput struct {
	TeamID string
	Email  string
	Role   string
}

type MembershipCheckInput struct {
	TeamID     string
	UserID     string
	Permission string
}

type TeamDetails struct {
	Team    domain.Team
	Members []domain.TeamMember
	Invites []domain.Invite
}

type AcceptInviteResult struct {
	TeamID     string
	MemberRole string
	Status     string
}

type Service struct {
	cfg Config

	teams       ports.TeamRepository
	members     ports.TeamMemberRepository
	invites     ports.InviteRepository
	roles       ports.RolePolicyRepository
	auditLogs   ports.AuditLogRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	outbox      ports.OutboxRepository

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config Config

	Teams       ports.TeamRepository
	Members     ports.TeamMemberRepository
	Invites     ports.InviteRepository
	Roles       ports.RolePolicyRepository
	AuditLogs   ports.AuditLogRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
	Outbox      ports.OutboxRepository

	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M97-Team-Service"
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
	if cfg.InviteTTL <= 0 {
		cfg.InviteTTL = 7 * 24 * time.Hour
	}
	return &Service{
		cfg:          cfg,
		teams:        deps.Teams,
		members:      deps.Members,
		invites:      deps.Invites,
		roles:        deps.Roles,
		auditLogs:    deps.AuditLogs,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}

func normalizeActorRole(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "user"
	}
	return raw
}
