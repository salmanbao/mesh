package application

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

func (s *Service) ValidateToken(ctx context.Context, token string) (ports.AuthClaims, error) {
	if strings.TrimSpace(token) == "" {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	claims, err := s.authClient.ValidateToken(ctx, token)
	if err != nil {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	if !claims.Valid {
		return ports.AuthClaims{}, domain.ErrUnauthorized
	}
	return claims, nil
}

func (s *Service) GetUserIdentity(ctx context.Context, userID uuid.UUID) (domain.UserIdentity, error) {
	return s.authClient.GetUserIdentity(ctx, userID)
}
