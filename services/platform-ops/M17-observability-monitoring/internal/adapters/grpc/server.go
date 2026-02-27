package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/platform-ops/M17-observability-monitoring/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type ObservabilityInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewObservabilityInternalServer(service *application.Service) *ObservabilityInternalServer {
	return &ObservabilityInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *ObservabilityInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *ObservabilityInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *ObservabilityInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
