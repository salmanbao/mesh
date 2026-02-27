package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/platform-ops/M67-event-bus/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type EventBusInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewEventBusInternalServer(service *application.Service) *EventBusInternalServer {
	return &EventBusInternalServer{service: service}
}
func Register(server grpc.ServiceRegistrar, svc *EventBusInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}
func (s *EventBusInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
func (s *EventBusInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
