package grpc

import (
	"context"
	"strings"
	"time"

	financev1 "github.com/viralforge/mesh/contracts/gen/go/finance/v1"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M39-finance-service/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type FinanceInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	financev1.UnimplementedFinanceInternalServiceServer
	service *application.Service
}

func NewFinanceInternalServer(service *application.Service) *FinanceInternalServer {
	return &FinanceInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *FinanceInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
	financev1.RegisterFinanceInternalServiceServer(server, svc)
}

func (s *FinanceInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *FinanceInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}

func (s *FinanceInternalServer) CreateTransaction(ctx context.Context, req *financev1.CreateTransactionRequest) (*financev1.CreateTransactionResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), req.GetIdempotencyKey(), req.GetRequestId())
	tx, err := s.service.CreateTransaction(ctx, actor, application.CreateTransactionInput{
		UserID:                strings.TrimSpace(req.GetUserId()),
		CampaignID:            strings.TrimSpace(req.GetCampaignId()),
		ProductID:             strings.TrimSpace(req.GetProductId()),
		Provider:              domain.PaymentProvider(strings.ToLower(strings.TrimSpace(req.GetProvider()))),
		ProviderTransactionID: strings.TrimSpace(req.GetProviderTransactionId()),
		Amount:                req.GetAmount(),
		Currency:              strings.TrimSpace(req.GetCurrency()),
		TrafficSource:         strings.TrimSpace(req.GetTrafficSource()),
		UserTier:              strings.TrimSpace(req.GetUserTier()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &financev1.CreateTransactionResponse{Transaction: marshalTransaction(tx)}, nil
}

func (s *FinanceInternalServer) GetTransaction(ctx context.Context, req *financev1.GetTransactionRequest) (*financev1.GetTransactionResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), "", req.GetRequestId())
	tx, err := s.service.GetTransaction(ctx, actor, strings.TrimSpace(req.GetTransactionId()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &financev1.GetTransactionResponse{Transaction: marshalTransaction(tx)}, nil
}

func (s *FinanceInternalServer) ListTransactions(ctx context.Context, req *financev1.ListTransactionsRequest) (*financev1.ListTransactionsResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), "", req.GetRequestId())
	out, err := s.service.ListTransactions(ctx, actor, ports.TransactionListQuery{
		UserID: strings.TrimSpace(req.GetUserId()),
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &financev1.ListTransactionsResponse{
		Pagination: &financev1.Pagination{
			Limit:  int32(out.Pagination.Limit),
			Offset: int32(out.Pagination.Offset),
			Total:  int64(out.Pagination.Total),
		},
	}
	for _, tx := range out.Items {
		resp.Items = append(resp.Items, marshalTransaction(tx))
	}
	return resp, nil
}

func (s *FinanceInternalServer) GetBalance(ctx context.Context, req *financev1.GetBalanceRequest) (*financev1.GetBalanceResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), "", req.GetRequestId())
	bal, err := s.service.GetBalance(ctx, actor, strings.TrimSpace(req.GetUserId()))
	if err != nil {
		return nil, toStatus(err)
	}
	return &financev1.GetBalanceResponse{Balance: marshalBalance(bal)}, nil
}

func (s *FinanceInternalServer) CreateRefund(ctx context.Context, req *financev1.CreateRefundRequest) (*financev1.CreateRefundResponse, error) {
	actor := actorFromRPC(req.GetSubjectId(), req.GetRole(), req.GetIdempotencyKey(), req.GetRequestId())
	ref, err := s.service.CreateRefund(ctx, actor, application.CreateRefundInput{
		TransactionID: strings.TrimSpace(req.GetTransactionId()),
		UserID:        strings.TrimSpace(req.GetUserId()),
		Amount:        req.GetAmount(),
		Reason:        strings.TrimSpace(req.GetReason()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	return &financev1.CreateRefundResponse{Refund: marshalRefund(ref)}, nil
}

func actorFromRPC(sub, role, idem, requestID string) application.Actor {
	sub = strings.TrimSpace(sub)
	role = strings.TrimSpace(role)
	if role == "" {
		role = "user"
	}
	return application.Actor{
		SubjectID:      sub,
		Role:           role,
		IdempotencyKey: strings.TrimSpace(idem),
		RequestID:      strings.TrimSpace(requestID),
	}
}

func marshalTransaction(t domain.Transaction) *financev1.Transaction {
	out := &financev1.Transaction{
		TransactionId:         t.TransactionID,
		UserId:                t.UserID,
		CampaignId:            t.CampaignID,
		ProductId:             t.ProductID,
		Provider:              string(t.Provider),
		ProviderTransactionId: t.ProviderTransactionID,
		Amount:                t.Amount,
		Currency:              strings.ToUpper(t.Currency),
		PlatformFeeRate:       t.PlatformFeeRate,
		Status:                string(t.Status),
		FailureReason:         t.FailureReason,
		CreatedAt:             t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             t.UpdatedAt.Format(time.RFC3339),
	}
	if t.SucceededAt != nil {
		out.SucceededAt = t.SucceededAt.Format(time.RFC3339)
	}
	if t.FailedAt != nil {
		out.FailedAt = t.FailedAt.Format(time.RFC3339)
	}
	if t.RefundedAt != nil {
		out.RefundedAt = t.RefundedAt.Format(time.RFC3339)
	}
	return out
}

func marshalRefund(r domain.Refund) *financev1.Refund {
	return &financev1.Refund{
		RefundId:      r.RefundID,
		TransactionId: r.TransactionID,
		UserId:        r.UserID,
		Amount:        r.Amount,
		Currency:      strings.ToUpper(r.Currency),
		Reason:        r.Reason,
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
	}
}

func marshalBalance(b domain.UserBalance) *financev1.Balance {
	return &financev1.Balance{
		UserId:            b.UserID,
		AvailableBalance:  b.AvailableBalance,
		PendingBalance:    b.PendingBalance,
		ReservedBalance:   b.ReservedBalance,
		NegativeBalance:   b.NegativeBalance,
		Currency:          strings.ToUpper(b.Currency),
		LastTransactionId: b.LastTransactionID,
		UpdatedAt:         b.UpdatedAt.Format(time.RFC3339),
	}
}

func toStatus(err error) error {
	switch err {
	case nil:
		return nil
	case domain.ErrUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case domain.ErrForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case domain.ErrNotFound:
		return status.Error(codes.NotFound, err.Error())
	case domain.ErrInvalidInput:
		return status.Error(codes.InvalidArgument, err.Error())
	case domain.ErrIdempotencyRequired:
		return status.Error(codes.FailedPrecondition, err.Error())
	case domain.ErrIdempotencyConflict, domain.ErrConflict:
		return status.Error(codes.Aborted, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
