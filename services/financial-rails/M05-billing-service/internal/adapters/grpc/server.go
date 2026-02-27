package grpc

import (
	"context"

	billingv1 "github.com/viralforge/mesh/contracts/gen/go/billing/v1"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/application"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M05-billing-service/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BillingInternalServer struct {
	grpc_health_v1.UnimplementedHealthServer
	billingv1.UnimplementedBillingOwnerServiceServer
	service *application.Service
}

func NewBillingInternalServer(service *application.Service) *BillingInternalServer {
	return &BillingInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *BillingInternalServer) {
	grpc_health_v1.RegisterHealthServer(server, svc)
	billingv1.RegisterBillingOwnerServiceServer(server, svc)
}

func (s *BillingInternalServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *BillingInternalServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	_ = req
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}

func (s *BillingInternalServer) GetInvoice(ctx context.Context, req *billingv1.GetInvoiceRequest) (*billingv1.GetInvoiceResponse, error) {
	invoice, err := s.service.GetInvoice(ctx, application.Actor{SubjectID: "owner-api", Role: "admin"}, req.GetInvoiceId())
	if err != nil {
		return nil, err
	}
	return &billingv1.GetInvoiceResponse{Invoice: toProtoInvoice(invoice)}, nil
}

func (s *BillingInternalServer) ListInvoices(ctx context.Context, req *billingv1.ListInvoicesRequest) (*billingv1.ListInvoicesResponse, error) {
	query := ports.InvoiceQuery{Status: req.GetStatus()}
	actor := application.Actor{SubjectID: "owner-api", Role: "admin"}
	out, err := s.service.SearchInvoices(ctx, actor, query)
	if err != nil {
		return nil, err
	}
	items := make([]*billingv1.Invoice, 0, len(out.Invoices))
	for _, inv := range out.Invoices {
		items = append(items, toProtoInvoice(inv))
	}
	return &billingv1.ListInvoicesResponse{
		Invoices: items,
		Pagination: &billingv1.Pagination{
			Limit:  int32(out.Pagination.Limit),
			Offset: int32(out.Pagination.Offset),
			Total:  int32(out.Pagination.Total),
		},
	}, nil
}

func toProtoInvoice(inv domain.Invoice) *billingv1.Invoice {
	pInvoice := &billingv1.Invoice{
		InvoiceId:     inv.InvoiceID,
		InvoiceNumber: inv.InvoiceNumber,
		CustomerId:    inv.CustomerID,
		CustomerName:  inv.CustomerName,
		CustomerEmail: inv.CustomerEmail,
		InvoiceType:   inv.InvoiceType,
		Currency:      inv.Currency,
		Subtotal:      inv.Subtotal,
		Total:         inv.Total,
		Status:        string(inv.Status),
		PaymentStatus: string(inv.PaymentStatus),
		PaymentMethod: inv.PaymentMethod,
		PdfUrl:        inv.PDFURL,
	}
	if !inv.DueDate.IsZero() {
		pInvoice.DueDate = timestamppb.New(inv.DueDate)
	}
	if !inv.InvoiceDate.IsZero() {
		pInvoice.InvoiceDate = timestamppb.New(inv.InvoiceDate)
	}
	if inv.PaidDate != nil && !inv.PaidDate.IsZero() {
		pInvoice.PaidDate = timestamppb.New(*inv.PaidDate)
	}
	if !inv.CreatedAt.IsZero() {
		pInvoice.CreatedAt = timestamppb.New(inv.CreatedAt)
	}
	if !inv.UpdatedAt.IsZero() {
		pInvoice.UpdatedAt = timestamppb.New(inv.UpdatedAt)
	}
	if inv.BillingAddress.Line1 != "" {
		pInvoice.BillingAddress = &billingv1.Address{
			Line1:      inv.BillingAddress.Line1,
			City:       inv.BillingAddress.City,
			State:      inv.BillingAddress.State,
			PostalCode: inv.BillingAddress.PostalCode,
			Country:    inv.BillingAddress.Country,
		}
	}
	if inv.Tax.Amount != 0 || inv.Tax.Rate != 0 || inv.Tax.Jurisdiction != "" {
		pInvoice.Tax = &billingv1.TaxBreakdown{
			Amount:       inv.Tax.Amount,
			Rate:         inv.Tax.Rate,
			Jurisdiction: inv.Tax.Jurisdiction,
		}
	}
	if len(inv.LineItems) > 0 {
		pInvoice.LineItems = make([]*billingv1.InvoiceLineItem, 0, len(inv.LineItems))
		for _, li := range inv.LineItems {
			pInvoice.LineItems = append(pInvoice.LineItems, &billingv1.InvoiceLineItem{
				LineItemId:   li.LineItemID,
				Description:  li.Description,
				Quantity:     int32(li.Quantity),
				UnitPrice:    li.UnitPrice,
				Amount:       li.Amount,
				SourceType:   li.SourceType,
				SourceId:     li.SourceID,
				CurrencyCode: li.CurrencyCode,
			})
		}
	}
	return pInvoice
}
