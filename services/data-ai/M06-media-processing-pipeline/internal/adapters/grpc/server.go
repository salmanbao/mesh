package grpc

import (
	"context"
	"errors"

	mediav1 "github.com/viralforge/mesh/contracts/gen/go/media/v1"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type MediaInternalServer struct {
	mediav1.UnimplementedMediaInternalServiceServer
	grpc_health_v1.UnimplementedHealthServer
	service *application.Service
}

func NewMediaInternalServer(service *application.Service) *MediaInternalServer {
	return &MediaInternalServer{service: service}
}

func Register(server grpc.ServiceRegistrar, svc *MediaInternalServer) {
	mediav1.RegisterMediaInternalServiceServer(server, svc)
	grpc_health_v1.RegisterHealthServer(server, svc)
}

func (s *MediaInternalServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (s *MediaInternalServer) Watch(_ *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING})
}

func (s *MediaInternalServer) GetPreviewUrl(ctx context.Context, req *mediav1.GetPreviewUrlRequest) (*mediav1.GetPreviewUrlResponse, error) {
	out, err := s.service.GetPreviewURL(ctx, req.GetAssetId(), req.GetExpirySeconds())
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &mediav1.GetPreviewUrlResponse{PreviewUrl: out.PreviewURL, ExpiresAt: out.ExpiresAt}, nil
}

func (s *MediaInternalServer) GetAssetMetadata(ctx context.Context, req *mediav1.GetAssetMetadataRequest) (*mediav1.GetAssetMetadataResponse, error) {
	meta, err := s.service.GetAssetMetadata(ctx, req.GetAssetId())
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &mediav1.GetAssetMetadataResponse{
		AssetId:         meta.AssetID,
		ContentType:     meta.ContentType,
		FileSizeBytes:   meta.FileSizeBytes,
		Width:           meta.Width,
		Height:          meta.Height,
		DurationSeconds: meta.DurationSeconds,
		Codec:           meta.Codec,
	}, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
