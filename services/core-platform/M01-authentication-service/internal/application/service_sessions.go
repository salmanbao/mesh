package application

import (
	"context"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

// ListSessions returns current and historical sessions for the authenticated user.
func (s *Service) ListSessions(ctx context.Context, jwtToken string) ([]SessionItem, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	sessions, err := s.sessions.ListByUser(ctx, claims.UserID, 100, 0)
	if err != nil {
		return nil, err
	}

	result := make([]SessionItem, 0, len(sessions))
	for _, it := range sessions {
		result = append(result, toSessionItem(it, claims.SessionID))
	}
	return result, nil
}

// ListLoginHistory returns login attempts with pagination and optional time/status filters.
func (s *Service) ListLoginHistory(ctx context.Context, jwtToken string, q LoginHistoryQuery) ([]LoginHistoryItem, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}
	offset := (q.Page - 1) * q.Limit

	var since *time.Time
	if q.Days > 0 {
		t := s.nowFn().Add(-time.Duration(q.Days) * 24 * time.Hour)
		since = &t
	}

	attempts, err := s.loginAttempts.ListByUser(ctx, claims.UserID, q.Limit, offset, since, strings.ToUpper(strings.TrimSpace(q.Status)))
	if err != nil {
		return nil, err
	}

	result := make([]LoginHistoryItem, 0, len(attempts))
	for _, attempt := range attempts {
		result = append(result, LoginHistoryItem{
			ID:            attempt.ID,
			Timestamp:     attempt.AttemptAt,
			Status:        attempt.Status,
			FailureReason: attempt.FailureReason,
			IPAddress:     attempt.IPAddress,
			DeviceName:    attempt.DeviceName,
			DeviceOS:      attempt.DeviceOS,
		})
	}
	return result, nil
}
