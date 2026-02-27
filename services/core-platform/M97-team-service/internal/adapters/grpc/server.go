package grpc

import (
	"context"
	"strings"
	"time"

	teamv1 "github.com/viralforge/mesh/contracts/gen/go/team/v1"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type TeamInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	teamv1.UnimplementedTeamInternalServiceServer
	service *application.Service
}

func NewTeamInternalServer(service *application.Service) *TeamInternalServer {
	return &TeamInternalServer{service: service}
}
func Register(server grpc.ServiceRegistrar, svc *TeamInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
	teamv1.RegisterTeamInternalServiceServer(server, svc)
}
func (s *TeamInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
func (s *TeamInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}

func (s *TeamInternalServer) CreateTeam(ctx context.Context, req *teamv1.CreateTeamRequest) (*teamv1.CreateTeamResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), req.GetIdempotencyKey(), req.GetRequestId())
	team, err := s.service.CreateTeam(ctx, actor, application.CreateTeamInput{
		ScopeType: strings.TrimSpace(req.GetScopeType()),
		ScopeID:   strings.TrimSpace(req.GetScopeId()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &teamv1.CreateTeamResponse{
		TeamId:  team.TeamID,
		OwnerId: team.OwnerID,
		Status:  team.Status,
	}, nil
}

func (s *TeamInternalServer) GetTeam(ctx context.Context, req *teamv1.GetTeamRequest) (*teamv1.GetTeamResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), "", req.GetRequestId())
	details, err := s.service.GetTeamDetails(ctx, actor, strings.TrimSpace(req.GetTeamId()))
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &teamv1.GetTeamResponse{
		TeamId: details.Team.TeamID,
	}
	for _, m := range details.Members {
		if m.Status != "active" {
			continue
		}
		resp.Members = append(resp.Members, &teamv1.TeamMember{
			UserId:   m.UserID,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
		})
	}
	for _, inv := range details.Invites {
		resp.Invites = append(resp.Invites, &teamv1.Invite{
			InviteId:  inv.InviteID,
			Status:    inv.Status,
			Email:     inv.Email,
			Role:      inv.Role,
			ExpiresAt: inv.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	return resp, nil
}

func (s *TeamInternalServer) CreateInvite(ctx context.Context, req *teamv1.CreateInviteRequest) (*teamv1.CreateInviteResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), req.GetIdempotencyKey(), req.GetRequestId())
	invite, err := s.service.CreateInvite(ctx, actor, application.CreateInviteInput{
		TeamID: strings.TrimSpace(req.GetTeamId()),
		Email:  strings.TrimSpace(req.GetEmail()),
		Role:   strings.TrimSpace(req.GetInviteRole()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &teamv1.CreateInviteResponse{
		InviteId:  invite.InviteID,
		Status:    invite.Status,
		ExpiresAt: invite.ExpiresAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *TeamInternalServer) AcceptInvite(ctx context.Context, req *teamv1.AcceptInviteRequest) (*teamv1.AcceptInviteResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), req.GetIdempotencyKey(), req.GetRequestId())
	res, err := s.service.AcceptInvite(ctx, actor, strings.TrimSpace(req.GetInviteId()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &teamv1.AcceptInviteResponse{
		TeamId:     res.TeamID,
		MemberRole: res.MemberRole,
		Status:     res.Status,
	}, nil
}

func (s *TeamInternalServer) CheckMembership(ctx context.Context, req *teamv1.CheckMembershipRequest) (*teamv1.CheckMembershipResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), "", req.GetRequestId())
	res, err := s.service.CheckMembership(ctx, actor, application.MembershipCheckInput{
		TeamID:     strings.TrimSpace(req.GetTeamId()),
		UserID:     strings.TrimSpace(req.GetUserId()),
		Permission: strings.TrimSpace(req.GetPermission()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &teamv1.CheckMembershipResponse{Allowed: res.Allowed, Role: res.Role}, nil
}

func actorFromRPC(sub, role, idem, requestID string) application.Actor {
	sub = strings.TrimSpace(sub)
	role = strings.TrimSpace(role)
	if role == "" {
		role = "user"
	}
	return application.Actor{
		SubjectID:      sub,
		Role:           role,
		IdempotencyKey: strings.TrimSpace(idem),
		RequestID:      strings.TrimSpace(requestID),
	}
}

func toStatus(err error) error {
	switch err {
	case nil:
		return nil
	case domain.ErrUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case domain.ErrForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case domain.ErrNotFound:
		return status.Error(codes.NotFound, err.Error())
	case domain.ErrInvalidInput:
		return status.Error(codes.InvalidArgument, err.Error())
	case domain.ErrIdempotencyRequired:
		return status.Error(codes.FailedPrecondition, err.Error())
	case domain.ErrIdempotencyConflict, domain.ErrConflict:
		return status.Error(codes.Aborted, err.Error())
	case domain.ErrInviteExpired, domain.ErrInviteNotPending:
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
