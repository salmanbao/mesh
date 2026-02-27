package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type AnalyticsInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewAnalyticsInternalServer(service *application.Service) *AnalyticsInternalServer {
	return &AnalyticsInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *AnalyticsInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *AnalyticsInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *AnalyticsInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
