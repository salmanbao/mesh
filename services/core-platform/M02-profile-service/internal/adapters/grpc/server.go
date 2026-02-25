package grpc

import (
	"context"

	"github.com/google/uuid"
	profilev1 "github.com/viralforge/mesh/contracts/gen/go/profile/v1"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProfileInternalServer struct {
	profilev1.UnimplementedProfileInternalServiceServer
	service *application.Service
}

func NewProfileInternalServer(service *application.Service) *ProfileInternalServer {
	return &ProfileInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *ProfileInternalServer) {
	profilev1.RegisterProfileInternalServiceServer(server, svc)
}

func (s *ProfileInternalServer) GetProfile(ctx context.Context, req *profilev1.GetProfileRequest) (*profilev1.GetProfileResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	profile, err := s.service.GetMyProfile(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}
	identity, _ := s.service.GetUserIdentity(ctx, userID)
	return &profilev1.GetProfileResponse{
		UserId:      profile.UserID,
		DisplayName: profile.DisplayName,
		AvatarUrl:   profile.AvatarURL,
		Role:        identity.Role,
		Status:      identity.Status,
	}, nil
}

func (s *ProfileInternalServer) BatchGetProfiles(ctx context.Context, req *profilev1.BatchGetProfilesRequest) (*profilev1.BatchGetProfilesResponse, error) {
	resp := &profilev1.BatchGetProfilesResponse{Profiles: make([]*profilev1.GetProfileResponse, 0, len(req.GetUserIds()))}
	for _, id := range req.GetUserIds() {
		item, err := s.GetProfile(ctx, &profilev1.GetProfileRequest{UserId: id})
		if err != nil {
			continue
		}
		resp.Profiles = append(resp.Profiles, item)
	}
	return resp, nil
}
