package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type CacheStateInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewCacheStateInternalServer(service *application.Service) *CacheStateInternalServer {
	return &CacheStateInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *CacheStateInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *CacheStateInternalServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *CacheStateInternalServer) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	return nil
}
