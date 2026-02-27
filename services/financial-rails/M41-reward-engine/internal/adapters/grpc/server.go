package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type RewardInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewRewardInternalServer(service *application.Service) *RewardInternalServer {
	return &RewardInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *RewardInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *RewardInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *RewardInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
