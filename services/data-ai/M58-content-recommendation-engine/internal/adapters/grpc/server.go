package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type RecommendationInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewRecommendationInternalServer(service *application.Service) *RecommendationInternalServer {
	return &RecommendationInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *RecommendationInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *RecommendationInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *RecommendationInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
