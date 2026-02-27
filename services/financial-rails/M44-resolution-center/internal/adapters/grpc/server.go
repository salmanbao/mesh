package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type ResolutionInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewResolutionInternalServer(service *application.Service) *ResolutionInternalServer {
	return &ResolutionInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *ResolutionInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *ResolutionInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *ResolutionInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
