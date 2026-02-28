package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/domain"
)

type Repositories struct {
	Tickets     *TicketRepository
	Replies     *ReplyRepository
	CSAT        *CSATRepository
	Agents      *AgentRepository
	Idempotency *IdempotencyRepository
}

func NewRepositories() *Repositories {
	agents := map[string]domain.Agent{
		"agent-billing":   {AgentID: "agent-billing", Role: "agent", SkillTags: []string{"billing", "refund"}, Active: true},
		"agent-technical": {AgentID: "agent-technical", Role: "agent", SkillTags: []string{"technical", "account", "other"}, Active: true},
		"agent-senior":    {AgentID: "agent-senior", Role: "senior_agent", SkillTags: []string{"billing", "technical", "refund", "account", "partner program", "other"}, Active: true},
		"manager-support": {AgentID: "manager-support", Role: "support_manager", SkillTags: []string{"management"}, Active: true},
	}
	return &Repositories{
		Tickets:     &TicketRepository{rows: map[string]domain.Ticket{}},
		Replies:     &ReplyRepository{rows: map[string][]domain.TicketReply{}},
		CSAT:        &CSATRepository{rows: map[string][]domain.CSATRating{}},
		Agents:      &AgentRepository{rows: agents},
		Idempotency: &IdempotencyRepository{rows: map[string]domain.IdempotencyRecord{}},
	}
}

type TicketRepository struct {
	mu   sync.Mutex
	rows map[string]domain.Ticket
}

func (r *TicketRepository) Create(_ context.Context, ticket domain.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[ticket.TicketID]; ok {
		return domain.ErrConflict
	}
	r.rows[ticket.TicketID] = ticket
	return nil
}

func (r *TicketRepository) Update(_ context.Context, ticket domain.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[ticket.TicketID]; !ok {
		return domain.ErrNotFound
	}
	r.rows[ticket.TicketID] = ticket
	return nil
}

func (r *TicketRepository) Get(_ context.Context, ticketID string) (domain.Ticket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ticket, ok := r.rows[strings.TrimSpace(ticketID)]
	if !ok {
		return domain.Ticket{}, domain.ErrNotFound
	}
	return ticket, nil
}

func (r *TicketRepository) Search(_ context.Context, filter domain.SearchFilter) ([]domain.Ticket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	category := strings.ToLower(strings.TrimSpace(filter.Category))
	userID := strings.TrimSpace(filter.UserID)
	assigned := strings.TrimSpace(filter.AssignedTo)
	items := make([]domain.Ticket, 0, len(r.rows))
	for _, ticket := range r.rows {
		if query != "" && !strings.Contains(strings.ToLower(ticket.Subject+" "+ticket.Description), query) {
			continue
		}
		if status != "" && strings.ToLower(ticket.Status) != status {
			continue
		}
		if category != "" && strings.ToLower(ticket.Category) != category {
			continue
		}
		if userID != "" && ticket.UserID != userID {
			continue
		}
		if assigned != "" && ticket.AssignedAgentID != assigned {
			continue
		}
		items = append(items, ticket)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]domain.Ticket, len(items))
	copy(out, items)
	return out, nil
}

type ReplyRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.TicketReply
}

func (r *ReplyRepository) Add(_ context.Context, reply domain.TicketReply) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[reply.TicketID] = append(r.rows[reply.TicketID], reply)
	return nil
}

func (r *ReplyRepository) ListByTicket(_ context.Context, ticketID string) ([]domain.TicketReply, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.rows[strings.TrimSpace(ticketID)]
	out := make([]domain.TicketReply, len(items))
	copy(out, items)
	return out, nil
}

type CSATRepository struct {
	mu   sync.Mutex
	rows map[string][]domain.CSATRating
}

func (r *CSATRepository) Add(_ context.Context, rating domain.CSATRating) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[rating.TicketID] = append(r.rows[rating.TicketID], rating)
	return nil
}

func (r *CSATRepository) ListByTicket(_ context.Context, ticketID string) ([]domain.CSATRating, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.rows[strings.TrimSpace(ticketID)]
	out := make([]domain.CSATRating, len(items))
	copy(out, items)
	return out, nil
}

type AgentRepository struct {
	mu   sync.Mutex
	rows map[string]domain.Agent
}

func (r *AgentRepository) List(_ context.Context) ([]domain.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.Agent, 0, len(r.rows))
	for _, agent := range r.rows {
		items = append(items, agent)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].OpenTicketCount == items[j].OpenTicketCount {
			return items[i].AgentID < items[j].AgentID
		}
		return items[i].OpenTicketCount < items[j].OpenTicketCount
	})
	return items, nil
}

func (r *AgentRepository) Get(_ context.Context, agentID string) (domain.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agent, ok := r.rows[strings.TrimSpace(agentID)]
	if !ok {
		return domain.Agent{}, domain.ErrNotFound
	}
	return agent, nil
}

func (r *AgentRepository) Upsert(_ context.Context, agent domain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[agent.AgentID] = agent
	return nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*domain.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.rows[strings.TrimSpace(key)]
	if !ok {
		return nil, nil
	}
	if now.After(rec.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	copy := rec
	return &copy, nil
}

func (r *IdempotencyRepository) Upsert(_ context.Context, rec domain.IdempotencyRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[rec.Key] = rec
	return nil
}
