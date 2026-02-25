package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

// RequestPasswordReset creates a one-time reset token when the user exists.
// It intentionally returns success for unknown users to avoid account enumeration.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}

	user, err := s.users.GetByEmail(ctx, normalized)
	if err != nil {
		// Do not leak whether user exists.
		return nil
	}

	rawToken := randomHex(32)
	tokenHash := hashToken(rawToken)
	now := s.nowFn()
	if err := s.recovery.CreatePasswordResetToken(ctx, user.UserID, tokenHash, now, now.Add(time.Hour)); err != nil {
		return err
	}
	return nil
}

// ResetPassword consumes a reset token and updates the user credential hash.
func (s *Service) ResetPassword(ctx context.Context, req PasswordResetRequest) error {
	if strings.TrimSpace(req.Token) == "" {
		return fmt.Errorf("%w: token is required", domain.ErrInvalidInput)
	}
	if err := domain.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	userID, err := s.recovery.ConsumePasswordResetToken(ctx, hashToken(req.Token), s.nowFn())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrUnauthorized
		}
		return err
	}

	passwordHash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	return s.credentials.UpdatePassword(ctx, userID, passwordHash, s.nowFn())
}

// RequestEmailVerification issues a one-time verification token for the authenticated user.
func (s *Service) RequestEmailVerification(ctx context.Context, jwtToken string) error {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return domain.ErrUnauthorized
	}

	now := s.nowFn()
	token := randomHex(32)
	tokenHash := hashToken(token)
	if err := s.recovery.CreateEmailVerificationToken(ctx, claims.UserID, tokenHash, now, now.Add(24*time.Hour)); err != nil {
		return err
	}
	return nil
}

// VerifyEmail consumes a verification token and marks email as verified.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("%w: token is required", domain.ErrInvalidInput)
	}
	userID, err := s.recovery.ConsumeEmailVerificationToken(ctx, hashToken(token), s.nowFn())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrUnauthorized
		}
		return err
	}
	return s.credentials.SetEmailVerified(ctx, userID, true, s.nowFn())
}
