package postgres

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/ports"
)

type Repositories struct {
	Teams       *TeamRepository
	Members     *TeamMemberRepository
	Invites     *InviteRepository
	Roles       *RolePolicyRepository
	AuditLogs   *AuditLogRepository
	Idempotency *IdempotencyRepository
	EventDedup  *EventDedupRepository
	Outbox      *OutboxRepository
}

func NewRepositories() *Repositories {
	now := time.Now().UTC()
	roles := &RolePolicyRepository{rows: map[string]domain.RolePolicy{}}
	_ = roles.Upsert(context.Background(), domain.RolePolicy{Role: "owner", Permissions: []string{"team.manage", "invite.manage", "member.manage", "member.view"}, CreatedAt: now})
	_ = roles.Upsert(context.Background(), domain.RolePolicy{Role: "admin", Permissions: []string{"invite.manage", "member.manage", "member.view"}, CreatedAt: now})
	_ = roles.Upsert(context.Background(), domain.RolePolicy{Role: "editor", Permissions: []string{"member.view"}, CreatedAt: now})
	_ = roles.Upsert(context.Background(), domain.RolePolicy{Role: "viewer", Permissions: []string{"member.view"}, CreatedAt: now})
	return &Repositories{
		Teams:       &TeamRepository{byID: map[string]domain.Team{}, byScope: map[string]string{}},
		Members:     &TeamMemberRepository{byID: map[string]domain.TeamMember{}, byTeamUser: map[string]string{}},
		Invites:     &InviteRepository{byID: map[string]domain.Invite{}, pendingByTeamEmail: map[string]string{}},
		Roles:       roles,
		AuditLogs:   &AuditLogRepository{rows: []domain.AuditLog{}},
		Idempotency: &IdempotencyRepository{rows: map[string]ports.IdempotencyRecord{}},
		EventDedup:  &EventDedupRepository{rows: map[string]eventDedupRow{}},
		Outbox:      &OutboxRepository{rows: map[string]ports.OutboxRecord{}, order: []string{}},
	}
}

func scopeKey(scopeType, scopeID string) string {
	return strings.ToLower(strings.TrimSpace(scopeType)) + ":" + strings.TrimSpace(scopeID)
}
func teamUserKey(teamID, userID string) string {
	return strings.TrimSpace(teamID) + ":" + strings.TrimSpace(userID)
}
func teamEmailKey(teamID, email string) string {
	return strings.TrimSpace(teamID) + ":" + strings.ToLower(strings.TrimSpace(email))
}

type TeamRepository struct {
	mu      sync.Mutex
	byID    map[string]domain.Team
	byScope map[string]string
}

func (r *TeamRepository) Create(_ context.Context, row domain.Team) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.TeamID]; ok {
		return domain.ErrConflict
	}
	sk := scopeKey(row.ScopeType, row.ScopeID)
	if _, ok := r.byScope[sk]; ok {
		return domain.ErrConflict
	}
	r.byID[row.TeamID] = row
	r.byScope[sk] = row.TeamID
	return nil
}
func (r *TeamRepository) GetByID(_ context.Context, teamID string) (domain.Team, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(teamID)]
	if !ok {
		return domain.Team{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *TeamRepository) GetByScope(_ context.Context, scopeType, scopeID string) (domain.Team, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byScope[scopeKey(scopeType, scopeID)]
	if !ok {
		return domain.Team{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok {
		return domain.Team{}, domain.ErrNotFound
	}
	return row, nil
}

type TeamMemberRepository struct {
	mu         sync.Mutex
	byID       map[string]domain.TeamMember
	byTeamUser map[string]string
}

func (r *TeamMemberRepository) Create(_ context.Context, row domain.TeamMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.TeamMemberID]; ok {
		return domain.ErrConflict
	}
	k := teamUserKey(row.TeamID, row.UserID)
	if id, ok := r.byTeamUser[k]; ok {
		existing := r.byID[id]
		if existing.Status == domain.MemberStatusActive {
			return domain.ErrConflict
		}
	}
	r.byID[row.TeamMemberID] = row
	r.byTeamUser[k] = row.TeamMemberID
	return nil
}
func (r *TeamMemberRepository) GetActiveByTeamUser(_ context.Context, teamID, userID string) (domain.TeamMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byTeamUser[teamUserKey(teamID, userID)]
	if !ok {
		return domain.TeamMember{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok || row.Status != domain.MemberStatusActive {
		return domain.TeamMember{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *TeamMemberRepository) ListByTeamID(_ context.Context, teamID string) ([]domain.TeamMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.TeamMember, 0)
	for _, row := range r.byID {
		if row.TeamID == strings.TrimSpace(teamID) {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out, nil
}

type InviteRepository struct {
	mu                 sync.Mutex
	byID               map[string]domain.Invite
	pendingByTeamEmail map[string]string
}

func (r *InviteRepository) Create(_ context.Context, row domain.Invite) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[row.InviteID]; ok {
		return domain.ErrConflict
	}
	pk := teamEmailKey(row.TeamID, row.Email)
	if id, ok := r.pendingByTeamEmail[pk]; ok {
		if existing, ok := r.byID[id]; ok && existing.Status == domain.InviteStatusPending {
			return domain.ErrConflict
		}
	}
	r.byID[row.InviteID] = row
	if row.Status == domain.InviteStatusPending {
		r.pendingByTeamEmail[pk] = row.InviteID
	}
	return nil
}
func (r *InviteRepository) GetByID(_ context.Context, inviteID string) (domain.Invite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.byID[strings.TrimSpace(inviteID)]
	if !ok {
		return domain.Invite{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *InviteRepository) FindPendingByTeamEmail(_ context.Context, teamID, email string) (domain.Invite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.pendingByTeamEmail[teamEmailKey(teamID, email)]
	if !ok {
		return domain.Invite{}, domain.ErrNotFound
	}
	row, ok := r.byID[id]
	if !ok || row.Status != domain.InviteStatusPending {
		return domain.Invite{}, domain.ErrNotFound
	}
	return row, nil
}
func (r *InviteRepository) Update(_ context.Context, row domain.Invite) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.byID[row.InviteID]
	if !ok {
		return domain.ErrNotFound
	}
	delete(r.pendingByTeamEmail, teamEmailKey(old.TeamID, old.Email))
	r.byID[row.InviteID] = row
	if row.Status == domain.InviteStatusPending {
		r.pendingByTeamEmail[teamEmailKey(row.TeamID, row.Email)] = row.InviteID
	}
	return nil
}
func (r *InviteRepository) ListByTeamID(_ context.Context, teamID string) ([]domain.Invite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Invite, 0)
	for _, row := range r.byID {
		if row.TeamID == strings.TrimSpace(teamID) {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

type RolePolicyRepository struct {
	mu   sync.Mutex
	rows map[string]domain.RolePolicy
}

func (r *RolePolicyRepository) List(_ context.Context) ([]domain.RolePolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.RolePolicy, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Role < out[j].Role })
	return out, nil
}
func (r *RolePolicyRepository) Upsert(_ context.Context, row domain.RolePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row.Role == "" {
		row.Role = uuid.NewString()
	}
	r.rows[row.Role] = row
	return nil
}

type AuditLogRepository struct {
	mu   sync.Mutex
	rows []domain.AuditLog
}

func (r *AuditLogRepository) Create(_ context.Context, row domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, row)
	return nil
}
func (r *AuditLogRepository) ListByTeamID(_ context.Context, teamID string, limit int) ([]domain.AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.AuditLog, 0)
	for i := len(r.rows) - 1; i >= 0; i-- {
		if r.rows[i].TeamID == strings.TrimSpace(teamID) {
			out = append(out, r.rows[i])
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

type IdempotencyRepository struct {
	mu   sync.Mutex
	rows map[string]ports.IdempotencyRecord
}

func (r *IdempotencyRepository) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok {
		return nil, nil
	}
	if now.After(row.ExpiresAt) {
		delete(r.rows, key)
		return nil, nil
	}
	copyRow := row
	copyRow.ResponseBody = append([]byte(nil), row.ResponseBody...)
	return &copyRow, nil
}
func (r *IdempotencyRepository) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.rows[key]; ok && time.Now().UTC().Before(existing.ExpiresAt) {
		return domain.ErrConflict
	}
	r.rows[key] = ports.IdempotencyRecord{Key: key, RequestHash: requestHash, ExpiresAt: expiresAt}
	return nil
}
func (r *IdempotencyRepository) Complete(_ context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[key]
	if !ok {
		return domain.ErrNotFound
	}
	row.ResponseCode = responseCode
	row.ResponseBody = append([]byte(nil), responseBody...)
	if at.After(row.ExpiresAt) {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	r.rows[key] = row
	return nil
}

type eventDedupRow struct {
	EventID   string
	EventType string
	ExpiresAt time.Time
}

type EventDedupRepository struct {
	mu   sync.Mutex
	rows map[string]eventDedupRow
}

func (r *EventDedupRepository) IsDuplicate(_ context.Context, eventID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[eventID]
	if !ok {
		return false, nil
	}
	if now.After(row.ExpiresAt) {
		delete(r.rows, eventID)
		return false, nil
	}
	return true, nil
}
func (r *EventDedupRepository) MarkProcessed(_ context.Context, eventID, eventType string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[eventID] = eventDedupRow{EventID: eventID, EventType: eventType, ExpiresAt: expiresAt}
	return nil
}

type OutboxRepository struct {
	mu    sync.Mutex
	rows  map[string]ports.OutboxRecord
	order []string
}

func (r *OutboxRepository) Enqueue(_ context.Context, row ports.OutboxRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[row.RecordID]; ok {
		return domain.ErrConflict
	}
	r.rows[row.RecordID] = row
	r.order = append(r.order, row.RecordID)
	return nil
}
func (r *OutboxRepository) ListPending(_ context.Context, limit int) ([]ports.OutboxRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]ports.OutboxRecord, 0, limit)
	for _, id := range r.order {
		row, ok := r.rows[id]
		if !ok || row.SentAt != nil {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
func (r *OutboxRepository) MarkSent(_ context.Context, recordID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[recordID]
	if !ok {
		return domain.ErrNotFound
	}
	row.SentAt = &at
	r.rows[recordID] = row
	return nil
}
