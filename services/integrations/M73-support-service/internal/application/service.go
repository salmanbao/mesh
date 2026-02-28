package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M73-support-service/internal/ports"
)

type Service struct {
	cfg         Config
	tickets     ports.TicketRepository
	replies     ports.ReplyRepository
	csat        ports.CSATRepository
	agents      ports.AgentRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

var idCounter uint64

type Dependencies struct {
	Config      Config
	Tickets     ports.TicketRepository
	Replies     ports.ReplyRepository
	CSAT        ports.CSATRepository
	Agents      ports.AgentRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	s := &Service{
		cfg:         deps.Config,
		tickets:     deps.Tickets,
		replies:     deps.Replies,
		csat:        deps.CSAT,
		agents:      deps.Agents,
		idempotency: deps.Idempotency,
		nowFn:       time.Now().UTC,
	}
	if s.cfg.IdempotencyTTL == 0 {
		s.cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return s
}

func (s *Service) CreateTicket(ctx context.Context, actor Actor, input CreateTicketInput) (domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Ticket{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Ticket{}, domain.ErrIdempotencyRequired
	}
	if err := validateCreateTicket(input); err != nil {
		return domain.Ticket{}, err
	}
	requestHash := hashJSON(input)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Ticket{}, err
	} else if ok {
		var ticket domain.Ticket
		_ = json.Unmarshal(rec, &ticket)
		return ticket, nil
	}
	category := normalizeCategory(input.Category)
	priority := normalizePriority(input.Priority)
	channel := strings.TrimSpace(input.Channel)
	if channel == "" {
		channel = "api"
	}
	now := s.nowFn()
	ticket := domain.Ticket{
		TicketID:         newTicketID(now),
		UserID:           strings.TrimSpace(actor.SubjectID),
		Subject:          strings.TrimSpace(input.Subject),
		Description:      strings.TrimSpace(input.Description),
		Category:         category,
		Priority:         priority,
		Status:           "open",
		SubStatus:        "new",
		Channel:          channel,
		EntityType:       strings.TrimSpace(input.EntityType),
		EntityID:         strings.TrimSpace(input.EntityID),
		SLAResponseDueAt: now.Add(slaDuration(priority)),
		LastActivityAt:   now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if agent, ok, err := s.pickAgent(ctx, category); err != nil {
		return domain.Ticket{}, err
	} else if ok {
		ticket.AssignedAgentID = agent.AgentID
		agent.OpenTicketCount++
		_ = s.agents.Upsert(ctx, agent)
	}
	if err := s.tickets.Create(ctx, ticket); err != nil {
		return domain.Ticket{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, ticket)
	return ticket, nil
}

func (s *Service) CreateTicketFromEmail(ctx context.Context, actor Actor, input CreateFromEmailInput) (domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Ticket{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Ticket{}, domain.ErrIdempotencyRequired
	}
	return s.CreateTicket(ctx, Actor{SubjectID: strings.TrimSpace(input.SenderEmail), IdempotencyKey: actor.IdempotencyKey}, CreateTicketInput{
		Subject:     input.Subject,
		Description: input.Description,
		Category:    "Other",
		Priority:    "normal",
		Channel:     "email",
	})
}

func (s *Service) GetTicket(ctx context.Context, actor Actor, ticketID string) (domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Ticket{}, domain.ErrUnauthorized
	}
	return s.tickets.Get(ctx, ticketID)
}

func (s *Service) SearchTickets(ctx context.Context, actor Actor, input SearchTicketsInput) ([]domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.tickets.Search(ctx, domain.SearchFilter{
		Query:      input.Query,
		Status:     input.Status,
		Category:   input.Category,
		UserID:     input.UserID,
		AssignedTo: input.AssignedTo,
		Limit:      input.Limit,
	})
}

func (s *Service) UpdateTicket(ctx context.Context, actor Actor, ticketID string, input UpdateTicketInput) (domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Ticket{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Ticket{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashJSON(map[string]any{"ticket_id": strings.TrimSpace(ticketID), "body": input})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Ticket{}, err
	} else if ok {
		var ticket domain.Ticket
		_ = json.Unmarshal(rec, &ticket)
		return ticket, nil
	}
	ticket, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !ticket.DeletedAt.IsZero() || ticket.Status == "deleted" {
		return domain.Ticket{}, domain.ErrNotFound
	}
	if status := strings.TrimSpace(input.Status); status != "" {
		status = normalizeTicketStatus(status)
		if status == "" {
			return domain.Ticket{}, domain.ErrInvalidInput
		}
		ticket.Status = status
		if status == "closed" {
			ticket.ClosedAt = s.nowFn()
		}
	}
	if sub := strings.TrimSpace(input.SubStatus); sub != "" {
		ticket.SubStatus = sub
	}
	if priority := strings.TrimSpace(input.Priority); priority != "" {
		ticket.Priority = normalizePriority(priority)
	}
	ticket.UpdatedAt = s.nowFn()
	ticket.LastActivityAt = ticket.UpdatedAt
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return domain.Ticket{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, ticket)
	return ticket, nil
}

func (s *Service) DeleteTicket(ctx context.Context, actor Actor, ticketID string) (domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Ticket{}, domain.ErrUnauthorized
	}
	ticket, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.Status != "deleted" {
		now := s.nowFn()
		ticket.Status = "deleted"
		ticket.DeletedAt = now
		ticket.UpdatedAt = now
		ticket.LastActivityAt = now
		if err := s.tickets.Update(ctx, ticket); err != nil {
			return domain.Ticket{}, err
		}
	}
	return ticket, nil
}

func (s *Service) AssignTicket(ctx context.Context, actor Actor, ticketID string, input AssignTicketInput) (domain.Ticket, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Ticket{}, domain.ErrUnauthorized
	}
	if !isManager(actor.Role) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Ticket{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.AgentID) == "" {
		return domain.Ticket{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"ticket_id": strings.TrimSpace(ticketID), "body": input})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Ticket{}, err
	} else if ok {
		var ticket domain.Ticket
		_ = json.Unmarshal(rec, &ticket)
		return ticket, nil
	}
	if _, err := s.agents.Get(ctx, input.AgentID); err != nil {
		return domain.Ticket{}, err
	}
	ticket, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return domain.Ticket{}, err
	}
	ticket.AssignedAgentID = strings.TrimSpace(input.AgentID)
	ticket.UpdatedAt = s.nowFn()
	ticket.LastActivityAt = ticket.UpdatedAt
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return domain.Ticket{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, ticket)
	return ticket, nil
}

func (s *Service) AddReply(ctx context.Context, actor Actor, ticketID string, input AddReplyInput) (domain.TicketReply, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.TicketReply{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.TicketReply{}, domain.ErrIdempotencyRequired
	}
	body := strings.TrimSpace(input.Body)
	if len(body) < 3 {
		return domain.TicketReply{}, domain.ErrInvalidInput
	}
	replyType := strings.ToLower(strings.TrimSpace(input.ReplyType))
	if replyType == "" {
		replyType = "public"
	}
	if replyType != "public" && replyType != "internal" {
		return domain.TicketReply{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"ticket_id": strings.TrimSpace(ticketID), "body": input})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.TicketReply{}, err
	} else if ok {
		var reply domain.TicketReply
		_ = json.Unmarshal(rec, &reply)
		return reply, nil
	}
	ticket, err := s.tickets.Get(ctx, ticketID)
	if err != nil {
		return domain.TicketReply{}, err
	}
	now := s.nowFn()
	reply := domain.TicketReply{
		ReplyID:   newID("rpl"),
		TicketID:  ticket.TicketID,
		AuthorID:  actor.SubjectID,
		ReplyType: replyType,
		Body:      body,
		CreatedAt: now,
	}
	if err := s.replies.Add(ctx, reply); err != nil {
		return domain.TicketReply{}, err
	}
	if ticket.Status == "resolved" && !isAgent(actor.Role) {
		ticket.Status = "open"
		ticket.SubStatus = "new"
	}
	if isAgent(actor.Role) && ticket.FirstResponseAt.IsZero() && replyType == "public" {
		ticket.FirstResponseAt = now
		ticket.SubStatus = "awaiting_response"
	} else if !isAgent(actor.Role) {
		ticket.SubStatus = "new"
	}
	ticket.LastActivityAt = now
	ticket.UpdatedAt = now
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return domain.TicketReply{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, reply)
	return reply, nil
}

func (s *Service) SubmitCSAT(ctx context.Context, actor Actor, ticketID string, input SubmitCSATInput) (domain.CSATRating, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CSATRating{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CSATRating{}, domain.ErrIdempotencyRequired
	}
	if input.Rating < 1 || input.Rating > 5 {
		return domain.CSATRating{}, domain.ErrInvalidInput
	}
	if _, err := s.tickets.Get(ctx, ticketID); err != nil {
		return domain.CSATRating{}, err
	}
	requestHash := hashJSON(map[string]any{"ticket_id": strings.TrimSpace(ticketID), "body": input})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CSATRating{}, err
	} else if ok {
		var rating domain.CSATRating
		_ = json.Unmarshal(rec, &rating)
		return rating, nil
	}
	rating := domain.CSATRating{
		CSATID:          newID("csat"),
		TicketID:        strings.TrimSpace(ticketID),
		UserID:          strings.TrimSpace(actor.SubjectID),
		Rating:          input.Rating,
		FeedbackComment: strings.TrimSpace(input.FeedbackComment),
		SubmittedAt:     s.nowFn(),
	}
	if err := s.csat.Add(ctx, rating); err != nil {
		return domain.CSATRating{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, rating)
	return rating, nil
}

func (s *Service) getIdempotent(ctx context.Context, key, hash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != hash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	return rec.Response, true, nil
}

func (s *Service) completeIdempotent(ctx context.Context, key, requestHash string, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.idempotency.Upsert(ctx, domain.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Response:    raw,
		ExpiresAt:   s.nowFn().Add(s.cfg.IdempotencyTTL),
	})
}

func (s *Service) pickAgent(ctx context.Context, category string) (domain.Agent, bool, error) {
	if s.agents == nil {
		return domain.Agent{}, false, nil
	}
	agents, err := s.agents.List(ctx)
	if err != nil {
		return domain.Agent{}, false, err
	}
	want := strings.ToLower(strings.TrimSpace(category))
	for _, agent := range agents {
		if !agent.Active || agent.OpenTicketCount >= 20 {
			continue
		}
		for _, skill := range agent.SkillTags {
			if strings.ToLower(strings.TrimSpace(skill)) == want {
				return agent, true, nil
			}
		}
	}
	for _, agent := range agents {
		if agent.Active && agent.OpenTicketCount < 20 && strings.Contains(strings.ToLower(agent.Role), "agent") {
			return agent, true, nil
		}
	}
	return domain.Agent{}, false, nil
}

func validateCreateTicket(input CreateTicketInput) error {
	subjectLen := len(strings.TrimSpace(input.Subject))
	descriptionLen := len(strings.TrimSpace(input.Description))
	if subjectLen < 3 || subjectLen > 200 || descriptionLen < 10 || descriptionLen > 5000 {
		return domain.ErrInvalidInput
	}
	if normalizeCategory(input.Category) == "" {
		return domain.ErrInvalidInput
	}
	return nil
}

func normalizeCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "billing":
		return "Billing"
	case "technical":
		return "Technical"
	case "refund":
		return "Refund"
	case "account":
		return "Account"
	case "partner program":
		return "Partner Program"
	case "other", "":
		return "Other"
	default:
		return ""
	}
}

func normalizePriority(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "low":
		return "low"
	case "normal", "":
		return "normal"
	default:
		return "normal"
	}
}

func normalizeTicketStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "open", "pending", "on_hold", "resolved", "closed", "deleted":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func slaDuration(priority string) time.Duration {
	switch normalizePriority(priority) {
	case "critical":
		return 4 * time.Hour
	case "high":
		return 12 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func isManager(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == "admin" || role == "support_manager" || role == "team_lead"
}

func isAgent(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == "agent" || role == "senior_agent" || isManager(role)
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func newTicketID(now time.Time) string {
	return fmt.Sprintf("TKT-%s-%s", now.Format("20060102"), shortID(nextIDSeed(now)))
}

func newID(prefix string) string {
	return prefix + "-" + shortID(nextIDSeed(time.Now().UTC()))
}

func nextIDSeed(now time.Time) int64 {
	n := atomic.AddUint64(&idCounter, 1)
	return now.UnixNano() + int64(n)
}

func shortID(v int64) string {
	if v < 0 {
		v = -v
	}
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 16)
	for v > 0 {
		buf = append(buf, chars[v%int64(len(chars))])
		v /= int64(len(chars))
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
