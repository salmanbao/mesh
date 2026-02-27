package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type SocialIntegrationVerificationInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewSocialIntegrationVerificationInternalServer(service *application.Service) *SocialIntegrationVerificationInternalServer {
	return &SocialIntegrationVerificationInternalServer{service: service}
}
func Register(server grpc.ServiceRegistrar, svc *SocialIntegrationVerificationInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}
func (s *SocialIntegrationVerificationInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service; _ = ctx; _ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
func (s *SocialIntegrationVerificationInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service; _ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
