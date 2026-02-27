package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
)

func (s *Service) CreateTeam(ctx context.Context, actor Actor, input CreateTeamInput) (domain.Team, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Team{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Team{}, domain.ErrIdempotencyRequired
	}
	input.ScopeType = strings.ToLower(strings.TrimSpace(input.ScopeType))
	input.ScopeID = strings.TrimSpace(input.ScopeID)
	if !domain.IsValidScopeType(input.ScopeType) || input.ScopeID == "" {
		return domain.Team{}, domain.ErrInvalidInput
	}

	requestHash := hashPayload(map[string]any{"op": "create_team", "actor": actor.SubjectID, "scope_type": input.ScopeType, "scope_id": input.ScopeID})
	if raw, ok, err := s.getIdempotentBody(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Team{}, err
	} else if ok {
		var cached domain.Team
		if json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Team{}, err
	}

	if _, err := s.teams.GetByScope(ctx, input.ScopeType, input.ScopeID); err == nil {
		return domain.Team{}, domain.ErrConflict
	}

	now := s.nowFn()
	team := domain.Team{
		TeamID:    uuid.NewString(),
		ScopeType: input.ScopeType,
		ScopeID:   input.ScopeID,
		OwnerID:   actor.SubjectID,
		Status:    domain.TeamStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.teams.Create(ctx, team); err != nil {
		return domain.Team{}, err
	}
	ownerMember := domain.TeamMember{
		TeamMemberID: uuid.NewString(),
		TeamID:       team.TeamID,
		UserID:       actor.SubjectID,
		Role:         "owner",
		Status:       domain.MemberStatusActive,
		JoinedAt:     now,
	}
	if err := s.members.Create(ctx, ownerMember); err != nil {
		return domain.Team{}, err
	}
	_ = s.appendAudit(ctx, team.TeamID, actor.SubjectID, "team.created", actor.SubjectID, map[string]string{"scope_type": team.ScopeType, "scope_id": team.ScopeID})
	_ = s.enqueueTeamCreated(ctx, team, actor.RequestID, now)
	_ = s.enqueueTeamMemberAdded(ctx, team.TeamID, actor.SubjectID, "owner", actor.RequestID, now)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, team)
	return team, nil
}

func (s *Service) GetTeamDetails(ctx context.Context, actor Actor, teamID string) (TeamDetails, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return TeamDetails{}, domain.ErrUnauthorized
	}
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return TeamDetails{}, domain.ErrInvalidInput
	}
	if !isPrivilegedActor(actor) {
		if _, err := s.requireTeamPermission(ctx, actor, teamID, "member.view"); err != nil {
			return TeamDetails{}, err
		}
	}
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		return TeamDetails{}, err
	}
	members, err := s.members.ListByTeamID(ctx, teamID)
	if err != nil {
		return TeamDetails{}, err
	}
	invites, err := s.invites.ListByTeamID(ctx, teamID)
	if err != nil {
		return TeamDetails{}, err
	}
	return TeamDetails{Team: team, Members: members, Invites: invites}, nil
}

func (s *Service) CreateInvite(ctx context.Context, actor Actor, input CreateInviteInput) (domain.Invite, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Invite{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Invite{}, domain.ErrIdempotencyRequired
	}
	input.TeamID = strings.TrimSpace(input.TeamID)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Role = strings.ToLower(strings.TrimSpace(input.Role))
	if input.TeamID == "" || input.Email == "" || !strings.Contains(input.Email, "@") || !domain.IsValidRole(input.Role) {
		return domain.Invite{}, domain.ErrInvalidInput
	}
	if _, err := s.requireTeamPermission(ctx, actor, input.TeamID, "invite.manage"); err != nil {
		return domain.Invite{}, err
	}
	if _, err := s.teams.GetByID(ctx, input.TeamID); err != nil {
		return domain.Invite{}, err
	}

	requestHash := hashPayload(map[string]any{"op": "create_invite", "actor": actor.SubjectID, "team_id": input.TeamID, "email": input.Email, "role": input.Role})
	if raw, ok, err := s.getIdempotentBody(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Invite{}, err
	} else if ok {
		var cached domain.Invite
		if json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Invite{}, err
	}
	if _, err := s.invites.FindPendingByTeamEmail(ctx, input.TeamID, input.Email); err == nil {
		return domain.Invite{}, domain.ErrConflict
	}

	now := s.nowFn()
	invite := domain.Invite{
		InviteID:  uuid.NewString(),
		TeamID:    input.TeamID,
		Email:     input.Email,
		Role:      input.Role,
		Status:    domain.InviteStatusPending,
		InvitedBy: actor.SubjectID,
		ExpiresAt: now.Add(s.cfg.InviteTTL),
		Token:     uuid.NewString(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.invites.Create(ctx, invite); err != nil {
		return domain.Invite{}, err
	}
	_ = s.appendAudit(ctx, invite.TeamID, actor.SubjectID, "team.invite.sent", "", map[string]string{"invite_id": invite.InviteID, "email": invite.Email, "role": invite.Role})
	_ = s.enqueueTeamInviteSent(ctx, invite, actor.RequestID, now)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, invite)
	return invite, nil
}

func (s *Service) AcceptInvite(ctx context.Context, actor Actor, inviteID string) (AcceptInviteResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return AcceptInviteResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return AcceptInviteResult{}, domain.ErrIdempotencyRequired
	}
	inviteID = strings.TrimSpace(inviteID)
	if inviteID == "" {
		return AcceptInviteResult{}, domain.ErrInvalidInput
	}

	requestHash := hashPayload(map[string]any{"op": "accept_invite", "actor": actor.SubjectID, "invite_id": inviteID})
	if raw, ok, err := s.getIdempotentBody(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return AcceptInviteResult{}, err
	} else if ok {
		var cached AcceptInviteResult
		if json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return AcceptInviteResult{}, err
	}

	invite, err := s.invites.GetByID(ctx, inviteID)
	if err != nil {
		return AcceptInviteResult{}, err
	}
	now := s.nowFn()
	if invite.Status == domain.InviteStatusAccepted && strings.TrimSpace(invite.AcceptedBy) == strings.TrimSpace(actor.SubjectID) {
		member, err := s.members.GetActiveByTeamUser(ctx, invite.TeamID, actor.SubjectID)
		if err != nil {
			return AcceptInviteResult{}, err
		}
		res := AcceptInviteResult{TeamID: invite.TeamID, MemberRole: member.Role, Status: "accepted"}
		_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, res)
		return res, nil
	}
	if invite.Status != domain.InviteStatusPending {
		return AcceptInviteResult{}, domain.ErrInviteNotPending
	}
	if now.After(invite.ExpiresAt) {
		invite.Status = domain.InviteStatusExpired
		invite.UpdatedAt = now
		_ = s.invites.Update(ctx, invite)
		return AcceptInviteResult{}, domain.ErrInviteExpired
	}

	memberRole := invite.Role
	memberCreated := false
	if existing, err := s.members.GetActiveByTeamUser(ctx, invite.TeamID, actor.SubjectID); err == nil {
		memberRole = existing.Role
	} else {
		member := domain.TeamMember{
			TeamMemberID: uuid.NewString(),
			TeamID:       invite.TeamID,
			UserID:       actor.SubjectID,
			Role:         invite.Role,
			Status:       domain.MemberStatusActive,
			JoinedAt:     now,
		}
		if err := s.members.Create(ctx, member); err != nil {
			return AcceptInviteResult{}, err
		}
		memberCreated = true
	}

	invite.Status = domain.InviteStatusAccepted
	invite.AcceptedBy = actor.SubjectID
	invite.AcceptedAt = &now
	invite.UpdatedAt = now
	if err := s.invites.Update(ctx, invite); err != nil {
		return AcceptInviteResult{}, err
	}
	_ = s.appendAudit(ctx, invite.TeamID, actor.SubjectID, "team.invite.accepted", actor.SubjectID, map[string]string{"invite_id": invite.InviteID, "role": memberRole})
	_ = s.enqueueTeamInviteAccepted(ctx, invite.TeamID, invite.InviteID, actor.SubjectID, actor.RequestID, now)
	if memberCreated {
		_ = s.enqueueTeamMemberAdded(ctx, invite.TeamID, actor.SubjectID, memberRole, actor.RequestID, now)
	}

	res := AcceptInviteResult{TeamID: invite.TeamID, MemberRole: memberRole, Status: "accepted"}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, res)
	return res, nil
}

func (s *Service) CheckMembership(ctx context.Context, actor Actor, input MembershipCheckInput) (domain.MembershipResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.MembershipResult{}, domain.ErrUnauthorized
	}
	input.TeamID = strings.TrimSpace(input.TeamID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Permission = strings.TrimSpace(input.Permission)
	if input.TeamID == "" {
		return domain.MembershipResult{}, domain.ErrInvalidInput
	}
	if input.UserID == "" {
		input.UserID = actor.SubjectID
	}
	if !isPrivilegedActor(actor) && input.UserID != actor.SubjectID {
		return domain.MembershipResult{}, domain.ErrForbidden
	}

	member, err := s.members.GetActiveByTeamUser(ctx, input.TeamID, input.UserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.MembershipResult{Allowed: false}, nil
		}
		return domain.MembershipResult{}, err
	}
	return domain.MembershipResult{Allowed: s.roleAllows(ctx, member.Role, input.Permission), Role: member.Role}, nil
}

func (s *Service) appendAudit(ctx context.Context, teamID, actorUserID, action, targetUserID string, meta map[string]string) error {
	if s.auditLogs == nil {
		return nil
	}
	clean := map[string]string{}
	for k, v := range meta {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		clean[k] = v
	}
	return s.auditLogs.Create(ctx, domain.AuditLog{
		AuditID:      uuid.NewString(),
		TeamID:       strings.TrimSpace(teamID),
		ActorUserID:  strings.TrimSpace(actorUserID),
		Action:       strings.TrimSpace(action),
		TargetUserID: strings.TrimSpace(targetUserID),
		Metadata:     clean,
		OccurredAt:   s.nowFn(),
	})
}

func (s *Service) requireTeamPermission(ctx context.Context, actor Actor, teamID, permission string) (domain.TeamMember, error) {
	if isPrivilegedActor(actor) {
		return domain.TeamMember{}, nil
	}
	member, err := s.members.GetActiveByTeamUser(ctx, strings.TrimSpace(teamID), strings.TrimSpace(actor.SubjectID))
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.TeamMember{}, domain.ErrForbidden
		}
		return domain.TeamMember{}, err
	}
	if !s.roleAllows(ctx, member.Role, permission) {
		return domain.TeamMember{}, domain.ErrForbidden
	}
	return member, nil
}

func (s *Service) roleAllows(ctx context.Context, role, permission string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	permission = strings.TrimSpace(permission)
	if role == "" {
		return false
	}
	if permission == "" {
		return true
	}
	if s.roles != nil {
		policies, err := s.roles.List(ctx)
		if err == nil {
			for _, p := range policies {
				if strings.EqualFold(p.Role, role) {
					for _, perm := range p.Permissions {
						if strings.TrimSpace(perm) == permission {
							return true
						}
					}
					return false
				}
			}
		}
	}
	switch role {
	case "owner":
		return true
	case "admin":
		return permission == "invite.manage" || permission == "member.manage" || permission == "member.view"
	case "editor", "viewer":
		return permission == "member.view"
	default:
		return false
	}
}

func isPrivilegedActor(actor Actor) bool {
	switch normalizeActorRole(actor.Role) {
	case "admin", "system":
		return true
	default:
		return false
	}
}

func hashPayload(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) getIdempotentBody(ctx context.Context, key, requestHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != requestHash {
		_ = s.publishDLQIdempotencyConflict(ctx, key, "")
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return append([]byte(nil), rec.ResponseBody...), true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	err := s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
	if err == domain.ErrConflict {
		return domain.ErrIdempotencyConflict
	}
	return err
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.idempotency.Complete(ctx, key, code, b, s.nowFn())
}
