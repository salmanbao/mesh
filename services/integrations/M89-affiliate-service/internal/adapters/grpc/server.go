package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type AffiliateInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewAffiliateInternalServer(service *application.Service) *AffiliateInternalServer {
	return &AffiliateInternalServer{service: service}
}
func Register(server grpc.ServiceRegistrar, svc *AffiliateInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}
func (s *AffiliateInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
func (s *AffiliateInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
