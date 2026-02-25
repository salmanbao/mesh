package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	authv1 "github.com/viralforge/mesh/contracts/gen/go/auth/v1"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthInternalServiceClient
}

func NewAuthClient(ctx context.Context, endpoint string) (*AuthClient, error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial auth grpc: %w", err)
	}
	client := authv1.NewAuthInternalServiceClient(conn)
	if _, err := client.GetPublicKeys(ctx, &authv1.GetPublicKeysRequest{}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("health check auth grpc: %w", err)
	}
	return &AuthClient{conn: conn, client: client}, nil
}

func (a *AuthClient) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func (a *AuthClient) ValidateToken(ctx context.Context, token string) (ports.AuthClaims, error) {
	resp, err := a.client.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
	if err != nil {
		return ports.AuthClaims{}, err
	}
	return ports.AuthClaims{
		UserID: resp.GetUserId(),
		Email:  resp.GetEmail(),
		Role:   resp.GetRole(),
		Valid:  resp.GetValid(),
	}, nil
}

func (a *AuthClient) GetUserIdentity(ctx context.Context, userID uuid.UUID) (domain.UserIdentity, error) {
	resp, err := a.client.GetUserIdentity(ctx, &authv1.GetUserIdentityRequest{UserId: userID.String()})
	if err != nil {
		return domain.UserIdentity{}, err
	}
	parsedUserID, err := uuid.Parse(resp.GetUserId())
	if err != nil {
		parsedUserID = userID
	}
	return domain.UserIdentity{
		UserID: parsedUserID,
		Email:  resp.GetEmail(),
		Role:   resp.GetRole(),
		Status: resp.GetStatus(),
	}, nil
}

var _ ports.AuthClient = (*AuthClient)(nil)
