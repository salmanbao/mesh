package ports

import (
	"context"
	"time"

	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/domain"
)

type TicketRepository interface {
	Create(ctx context.Context, ticket domain.Ticket) error
	Update(ctx context.Context, ticket domain.Ticket) error
	Get(ctx context.Context, ticketID string) (domain.Ticket, error)
	Search(ctx context.Context, filter domain.SearchFilter) ([]domain.Ticket, error)
}

type ReplyRepository interface {
	Add(ctx context.Context, reply domain.TicketReply) error
	ListByTicket(ctx context.Context, ticketID string) ([]domain.TicketReply, error)
}

type CSATRepository interface {
	Add(ctx context.Context, rating domain.CSATRating) error
	ListByTicket(ctx context.Context, ticketID string) ([]domain.CSATRating, error)
}

type AgentRepository interface {
	List(ctx context.Context) ([]domain.Agent, error)
	Get(ctx context.Context, agentID string) (domain.Agent, error)
	Upsert(ctx context.Context, agent domain.Agent) error
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error)
	Upsert(ctx context.Context, rec domain.IdempotencyRecord) error
}
