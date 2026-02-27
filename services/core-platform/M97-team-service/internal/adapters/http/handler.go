package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/contracts"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) createTeam(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	team, err := h.service.CreateTeam(r.Context(), actor, application.CreateTeamInput{ScopeType: strings.TrimSpace(req.ScopeType), ScopeID: strings.TrimSpace(req.ScopeID)})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateTeamResponse{TeamID: team.TeamID, OwnerID: team.OwnerID, Status: team.Status})
}

func (h *Handler) getTeam(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	details, err := h.service.GetTeamDetails(r.Context(), actor, chi.URLParam(r, "team_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	resp := contracts.TeamDetailsResponse{TeamID: details.Team.TeamID, Members: make([]contracts.TeamMemberDTO, 0, len(details.Members)), Invites: make([]contracts.InviteDTO, 0, len(details.Invites))}
	for _, m := range details.Members {
		if m.Status != "active" {
			continue
		}
		resp.Members = append(resp.Members, contracts.TeamMemberDTO{UserID: m.UserID, Role: m.Role, JoinedAt: m.JoinedAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
	}
	for _, inv := range details.Invites {
		resp.Invites = append(resp.Invites, contracts.InviteDTO{InviteID: inv.InviteID, Status: inv.Status, Email: inv.Email, Role: inv.Role, ExpiresAt: inv.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
	}
	writeSuccess(w, http.StatusOK, "", resp)
}

func (h *Handler) createInvite(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	var req contracts.CreateInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error(), requestIDFromContext(r.Context()))
		return
	}
	invite, err := h.service.CreateInvite(r.Context(), actor, application.CreateInviteInput{TeamID: chi.URLParam(r, "team_id"), Email: req.Email, Role: req.Role})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusCreated, "", contracts.CreateInviteResponse{InviteID: invite.InviteID, Status: invite.Status, ExpiresAt: invite.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	res, err := h.service.AcceptInvite(r.Context(), actor, chi.URLParam(r, "invite_id"))
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.AcceptInviteResponse{TeamID: res.TeamID, MemberRole: res.MemberRole, Status: res.Status})
}

func (h *Handler) checkMembership(w http.ResponseWriter, r *http.Request) {
	actor := actorFromContext(r.Context())
	res, err := h.service.CheckMembership(r.Context(), actor, application.MembershipCheckInput{
		TeamID:     strings.TrimSpace(r.URL.Query().Get("team_id")),
		UserID:     strings.TrimSpace(r.URL.Query().Get("user_id")),
		Permission: strings.TrimSpace(r.URL.Query().Get("permission")),
	})
	if err != nil {
		status, code := mapDomainError(err)
		writeError(w, status, code, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeSuccess(w, http.StatusOK, "", contracts.MembershipResponse{Allowed: res.Allowed, Role: res.Role})
}
