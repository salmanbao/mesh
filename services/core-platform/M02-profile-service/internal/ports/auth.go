package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
)

type AuthClaims struct {
	UserID string
	Email  string
	Role   string
	Valid  bool
}

type AuthClient interface {
	ValidateToken(ctx context.Context, token string) (AuthClaims, error)
	GetUserIdentity(ctx context.Context, userID uuid.UUID) (domain.UserIdentity, error)
}
