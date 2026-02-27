package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type PayoutInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewPayoutInternalServer(service *application.Service) *PayoutInternalServer {
	return &PayoutInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *PayoutInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *PayoutInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *PayoutInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
