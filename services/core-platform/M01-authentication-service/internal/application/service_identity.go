package application

import (
	"context"

	"github.com/google/uuid"
)

// GetUserIdentity exposes an owner-api read model for dependent services.
// It intentionally returns a narrow projection to preserve M01 data ownership.
func (s *Service) GetUserIdentity(ctx context.Context, userID uuid.UUID) (UserIdentity, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return UserIdentity{}, err
	}

	status := "active"
	if !user.IsActive {
		status = "disabled"
	}
	if user.DeletedAt != nil {
		status = "deleted"
	}

	return UserIdentity{
		UserID: user.UserID,
		Email:  user.Email,
		Role:   user.RoleName,
		Status: status,
	}, nil
}
