package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type BillingInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewBillingInternalServer(service *application.Service) *BillingInternalServer {
	return &BillingInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *BillingInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *BillingInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *BillingInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
