package grpc

import (
	"context"

	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/application"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type EscrowLedgerInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}
func NewEscrowLedgerInternalServer(service *application.Service) *EscrowLedgerInternalServer { return &EscrowLedgerInternalServer{service: service} }
func Register(server grpc.ServiceRegistrar, svc *EscrowLedgerInternalServer) { grpc_health_v1.RegisterHealthServer(server, svc) }
func (s *EscrowLedgerInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) { _ = s.service; _ = ctx; _ = req; return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil }
func (s *EscrowLedgerInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error { _ = s.service; _ = req; return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}) }
