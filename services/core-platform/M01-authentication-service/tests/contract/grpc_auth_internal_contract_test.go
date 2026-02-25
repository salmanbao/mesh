package contract

import (
	"context"
	"testing"

	authv1 "github.com/viralforge/mesh/contracts/gen/go/auth/v1"
	grpcadapter "github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/adapters/grpc"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthInternalValidateTokenContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newOIDCContractService()

	_, err := svc.Register(ctx, application.RegisterRequest{
		Email:         "grpc-contract@example.com",
		Password:      "StrongPass123!",
		Role:          "EDITOR",
		TermsAccepted: true,
	}, "")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	loginRes, err := svc.Login(ctx, application.LoginRequest{
		Email:      "grpc-contract@example.com",
		Password:   "StrongPass123!",
		IPAddress:  "127.0.0.1",
		UserAgent:  "grpc-contract-test",
		DeviceName: "test",
		DeviceOS:   "linux",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	server := grpcadapter.NewAuthInternalServer(svc)
	resp, err := server.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: loginRes.Token})
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}

	if !resp.GetValid() {
		t.Fatalf("expected valid token response")
	}
	if resp.GetEmail() != "grpc-contract@example.com" {
		t.Fatalf("unexpected email in response: %s", resp.GetEmail())
	}
}

func TestAuthInternalValidateTokenRejectsMissingToken(t *testing.T) {
	t.Parallel()

	server := grpcadapter.NewAuthInternalServer(newOIDCContractService())
	_, err := server.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{})
	if err == nil {
		t.Fatalf("expected invalid argument error")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %s", status.Code(err))
	}
}

func TestAuthInternalGetPublicKeysContract(t *testing.T) {
	t.Parallel()

	server := grpcadapter.NewAuthInternalServer(newOIDCContractService())
	resp, err := server.GetPublicKeys(context.Background(), &authv1.GetPublicKeysRequest{})
	if err != nil {
		t.Fatalf("get public keys failed: %v", err)
	}
	if len(resp.GetKeys()) == 0 {
		t.Fatalf("expected at least one JWK")
	}
	if resp.GetKeys()[0].GetKid() == "" {
		t.Fatalf("expected JWK kid to be populated")
	}
}

func TestAuthInternalGetUserIdentityContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newOIDCContractService()
	regRes, err := svc.Register(ctx, application.RegisterRequest{
		Email:         "identity-contract@example.com",
		Password:      "StrongPass123!",
		Role:          "INFLUENCER",
		TermsAccepted: true,
	}, "")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	server := grpcadapter.NewAuthInternalServer(svc)
	resp, err := server.GetUserIdentity(ctx, &authv1.GetUserIdentityRequest{UserId: regRes.UserID.String()})
	if err != nil {
		t.Fatalf("get user identity failed: %v", err)
	}

	if resp.GetUserId() != regRes.UserID.String() {
		t.Fatalf("unexpected user_id: %s", resp.GetUserId())
	}
	if resp.GetEmail() != "identity-contract@example.com" {
		t.Fatalf("unexpected email: %s", resp.GetEmail())
	}
	if resp.GetStatus() != "active" {
		t.Fatalf("unexpected status: %s", resp.GetStatus())
	}
}
