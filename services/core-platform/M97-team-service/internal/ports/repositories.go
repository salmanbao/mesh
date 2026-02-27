package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
)

type TeamRepository interface {
	Create(ctx context.Context, row domain.Team) error
	GetByID(ctx context.Context, teamID string) (domain.Team, error)
	GetByScope(ctx context.Context, scopeType, scopeID string) (domain.Team, error)
}

type TeamMemberRepository interface {
	Create(ctx context.Context, row domain.TeamMember) error
	GetActiveByTeamUser(ctx context.Context, teamID, userID string) (domain.TeamMember, error)
	ListByTeamID(ctx context.Context, teamID string) ([]domain.TeamMember, error)
}

type InviteRepository interface {
	Create(ctx context.Context, row domain.Invite) error
	GetByID(ctx context.Context, inviteID string) (domain.Invite, error)
	FindPendingByTeamEmail(ctx context.Context, teamID, email string) (domain.Invite, error)
	Update(ctx context.Context, row domain.Invite) error
	ListByTeamID(ctx context.Context, teamID string) ([]domain.Invite, error)
}

type RolePolicyRepository interface {
	List(ctx context.Context) ([]domain.RolePolicy, error)
	Upsert(ctx context.Context, row domain.RolePolicy) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, row domain.AuditLog) error
	ListByTeamID(ctx context.Context, teamID string, limit int) ([]domain.AuditLog, error)
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, record OutboxRecord) error
	ListPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkSent(ctx context.Context, recordID string, at time.Time) error
}
