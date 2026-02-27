package grpc

import (
	"context"

	escrowv1 "github.com/viralforge/mesh/contracts/gen/go/escrow/v1"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type EscrowLedgerInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	escrowv1.UnimplementedEscrowInternalServiceServer
	service *application.Service
}

func NewEscrowLedgerInternalServer(service *application.Service) *EscrowLedgerInternalServer {
	return &EscrowLedgerInternalServer{service: service}
}
func Register(server grpc.ServiceRegistrar, svc *EscrowLedgerInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
	escrowv1.RegisterEscrowInternalServiceServer(server, svc)
}

func (s *EscrowLedgerInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = s.service
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}
func (s *EscrowLedgerInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = s.service
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}

func (s *EscrowLedgerInternalServer) GetEscrowHold(ctx context.Context, req *escrowv1.GetEscrowHoldRequest) (*escrowv1.GetEscrowHoldResponse, error) {
	hold, err := s.service.GetEscrowHold(ctx, req.GetEscrowId())
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if err == domain.ErrInvalidInput {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, err
	}
	return &escrowv1.GetEscrowHoldResponse{Hold: toProtoHold(hold)}, nil
}

func (s *EscrowLedgerInternalServer) GetWalletBalance(ctx context.Context, req *escrowv1.GetWalletBalanceRequest) (*escrowv1.GetWalletBalanceResponse, error) {
	// owner_api read: bypass actor auth, rely on service mesh authN.
	bal, err := s.service.GetWalletBalance(ctx, application.Actor{SubjectID: "owner_api"}, req.GetCampaignId())
	if err != nil {
		if err == domain.ErrInvalidInput {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, err
	}
	return &escrowv1.GetWalletBalanceResponse{Balance: &escrowv1.WalletBalance{
		CampaignId:       bal.CampaignID,
		HeldBalance:      bal.HeldBalance,
		ReleasedBalance:  bal.ReleasedBalance,
		RefundedBalance:  bal.RefundedBalance,
		NetEscrowBalance: bal.NetEscrowBalance,
	}}, nil
}

func toProtoHold(h domain.EscrowHold) *escrowv1.EscrowHold {
	return &escrowv1.EscrowHold{
		EscrowId:        h.EscrowID,
		CampaignId:      h.CampaignID,
		CreatorId:       h.CreatorID,
		Status:          h.Status,
		OriginalAmount:  h.OriginalAmount,
		RemainingAmount: h.RemainingAmount,
		ReleasedAmount:  h.ReleasedAmount,
		RefundedAmount:  h.RefundedAmount,
	}
}
