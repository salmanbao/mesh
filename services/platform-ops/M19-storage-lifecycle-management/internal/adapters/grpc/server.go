package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type StorageLifecycleInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewStorageLifecycleInternalServer(service *application.Service) *StorageLifecycleInternalServer {
	return &StorageLifecycleInternalServer{service: service}
}
func Register(server grpc.ServiceRegistrar, svc *StorageLifecycleInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
}
func (s *StorageLifecycleInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
func (s *StorageLifecycleInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}
