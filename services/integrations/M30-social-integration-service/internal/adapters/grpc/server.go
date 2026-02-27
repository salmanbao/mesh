package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/integrations/M30-social-integration-service/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type SocialIntegrationInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewSocialIntegrationInternalServer(service *application.Service) *SocialIntegrationInternalServer {
	return &SocialIntegrationInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *SocialIntegrationInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *SocialIntegrationInternalServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *SocialIntegrationInternalServer) Watch(*grpc_health_v1.HealthCheckRequest, grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	return nil
}
