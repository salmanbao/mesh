package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	authv1 "github.com/viralforge/mesh/contracts/gen/go/auth/v1"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthInternalServer exposes M01 internal auth RPCs.
// This adapter keeps transport/error mapping concerns outside application logic.
type AuthInternalServer struct {
	authv1.UnimplementedAuthInternalServiceServer
	service *application.Service
}

// NewAuthInternalServer constructs the gRPC auth server adapter.
func NewAuthInternalServer(service *application.Service) *AuthInternalServer {
	return &AuthInternalServer{service: service}
}

// Register binds the auth internal service to a gRPC registrar.
func Register(server grpc.ServiceRegistrar, svc *AuthInternalServer) {
	authv1.RegisterAuthInternalServiceServer(server, svc)
}

func (s *AuthInternalServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	token := req.GetToken()
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "missing token")
	}

	claims, err := s.service.ValidateToken(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return &authv1.ValidateTokenResponse{
		Valid:     true,
		UserId:    claims.UserID.String(),
		Email:     claims.Email,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

func (s *AuthInternalServer) GetPublicKeys(ctx context.Context, _ *authv1.GetPublicKeysRequest) (*authv1.GetPublicKeysResponse, error) {
	keys, err := s.service.PublicJWKs()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get keys: %v", err)
	}

	out := make([]*authv1.JWK, 0, len(keys))
	for _, item := range keys {
		out = append(out, &authv1.JWK{
			Kid: valueAsString(item["kid"]),
			Kty: valueAsString(item["kty"]),
			Alg: valueAsString(item["alg"]),
			N:   valueAsString(item["n"]),
			E:   valueAsString(item["e"]),
		})
	}

	return &authv1.GetPublicKeysResponse{Keys: out}, nil
}

func (s *AuthInternalServer) GetUserIdentity(ctx context.Context, req *authv1.GetUserIdentityRequest) (*authv1.GetUserIdentityResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	identity, err := s.service.GetUserIdentity(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user identity: %v", err)
	}

	return &authv1.GetUserIdentityResponse{
		UserId: identity.UserID.String(),
		Email:  identity.Email,
		Role:   identity.Role,
		Status: identity.Status,
	}, nil
}

func valueAsString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}
