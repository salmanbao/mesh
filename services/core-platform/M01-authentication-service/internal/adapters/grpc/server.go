package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
)

type AuthInternalService interface {
	ValidateToken(context.Context, *structpb.Struct) (*structpb.Struct, error)
	GetPublicKeys(context.Context, *emptypb.Empty) (*structpb.Struct, error)
}

type AuthInternalServer struct {
	service *application.Service
}

func NewAuthInternalServer(service *application.Service) *AuthInternalServer {
	return &AuthInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc AuthInternalService) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "viralforge.auth.v1.AuthInternalService",
		HandlerType: (*AuthInternalService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "ValidateToken",
				Handler:    validateTokenHandler(svc),
			},
			{
				MethodName: "GetPublicKeys",
				Handler:    getPublicKeysHandler(svc),
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "mesh/contracts/proto/auth/v1/auth_internal.proto",
	}, svc)
}

func (s *AuthInternalServer) ValidateToken(ctx context.Context, req *structpb.Struct) (*structpb.Struct, error) {
	tokenVal := req.GetFields()["token"]
	if tokenVal == nil {
		return nil, status.Error(codes.InvalidArgument, "missing token")
	}
	token := tokenVal.GetStringValue()
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "missing token")
	}

	claims, err := s.service.ValidateToken(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	resp, err := structpb.NewStruct(map[string]any{
		"valid":      true,
		"user_id":    claims.UserID.String(),
		"email":      claims.Email,
		"role":       claims.Role,
		"expires_at": claims.ExpiresAt.Unix(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build response: %v", err)
	}
	return resp, nil
}

func (s *AuthInternalServer) GetPublicKeys(ctx context.Context, _ *emptypb.Empty) (*structpb.Struct, error) {
	keys, err := s.service.PublicJWKs()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get keys: %v", err)
	}
	resp, err := structpb.NewStruct(map[string]any{
		"keys": keys,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build response: %v", err)
	}
	return resp, nil
}

func validateTokenHandler(svc AuthInternalService) func(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error) {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		req := &structpb.Struct{}
		if err := dec(req); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return svc.ValidateToken(ctx, req)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/viralforge.auth.v1.AuthInternalService/ValidateToken",
		}
		handler := func(ctx context.Context, req any) (any, error) {
			typed, ok := req.(*structpb.Struct)
			if !ok {
				return nil, status.Error(codes.InvalidArgument, "invalid request type")
			}
			return svc.ValidateToken(ctx, typed)
		}
		return interceptor(ctx, req, info, handler)
	}
}

func getPublicKeysHandler(svc AuthInternalService) func(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error) {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		req := &emptypb.Empty{}
		if err := dec(req); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return svc.GetPublicKeys(ctx, req)
		}
		info := &grpc.UnaryServerInfo{
			Server:     srv,
			FullMethod: "/viralforge.auth.v1.AuthInternalService/GetPublicKeys",
		}
		handler := func(ctx context.Context, req any) (any, error) {
			typed, ok := req.(*emptypb.Empty)
			if !ok {
				return nil, status.Error(codes.InvalidArgument, "invalid request type")
			}
			return svc.GetPublicKeys(ctx, typed)
		}
		return interceptor(ctx, req, info, handler)
	}
}
