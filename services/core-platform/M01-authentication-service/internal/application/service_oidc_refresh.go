package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// RefreshExpiringOIDCTokens refreshes provider tokens that are expiring within the configured window.
func (s *Service) RefreshExpiringOIDCTokens(ctx context.Context, refreshWindow time.Duration, batchSize int) error {
	if s.oidcVerifier == nil {
		return nil
	}
	if refreshWindow <= 0 {
		refreshWindow = 24 * time.Hour
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	items, err := s.oidc.ListTokensExpiringBefore(ctx, s.nowFn().Add(refreshWindow), batchSize)
	if err != nil {
		return fmt.Errorf("list expiring oidc tokens: %w", err)
	}

	var firstErr error
	for _, item := range items {
		if strings.TrimSpace(item.RefreshToken) == "" {
			continue
		}
		if err := s.refreshOIDCTokenWithRetry(ctx, item); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Service) refreshOIDCTokenWithRetry(ctx context.Context, item ports.OIDCTokenRecord) error {
	backoff := time.Second
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		tokenSet, err := s.oidcVerifier.RefreshToken(ctx, item.Provider, item.RefreshToken)
		if err == nil {
			refreshToken := strings.TrimSpace(tokenSet.RefreshToken)
			if refreshToken == "" {
				refreshToken = item.RefreshToken
			}
			now := s.nowFn()
			if err := s.oidc.UpsertToken(ctx, item.UserID, item.Provider, tokenSet.AccessToken, refreshToken, tokenSet.ExpiresAt, now); err != nil {
				return fmt.Errorf("persist refreshed oidc token: %w", err)
			}
			_ = s.oidc.UpdateConnectionStatus(ctx, item.UserID, item.Provider, "ACTIVE", now)
			return nil
		}

		lastErr = err
		if attempt < 3 {
			slog.Default().WarnContext(ctx, "oidc token refresh attempt failed",
				"service", "M01-Authentication-Service",
				"module", "application",
				"layer", "application",
				"operation", "refresh_oidc_token",
				"outcome", "failure",
				"user_id", item.UserID,
				"provider", item.Provider,
				"attempt", attempt,
				"error", err,
			)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			backoff *= 2
			continue
		}
	}

	now := s.nowFn()
	_ = s.oidc.UpdateConnectionStatus(ctx, item.UserID, item.Provider, "EXPIRED", now)
	slog.Default().ErrorContext(ctx, "oidc token refresh failed after retries",
		"service", "M01-Authentication-Service",
		"module", "application",
		"layer", "application",
		"operation", "refresh_oidc_token",
		"outcome", "failure",
		"user_id", item.UserID,
		"provider", item.Provider,
		"error_code", "OIDC_REFRESH_FAILED",
		"error", lastErr,
	)
	if lastErr == nil {
		return errors.New("oidc token refresh failed")
	}
	return lastErr
}

func (s *Service) persistOIDCToken(ctx context.Context, userID uuid.UUID, provider string, identity ports.OIDCIdentity) error {
	accessToken := strings.TrimSpace(identity.AccessToken)
	if accessToken == "" {
		return nil
	}
	return s.oidc.UpsertToken(ctx, userID, provider, accessToken, strings.TrimSpace(identity.RefreshToken), identity.ExpiresAt, s.nowFn())
}
