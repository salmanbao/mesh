package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/data-ai/M55-dashboard-service/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type DashboardInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewDashboardInternalServer(service *application.Service) *DashboardInternalServer {
	return &DashboardInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *DashboardInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *DashboardInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *DashboardInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
