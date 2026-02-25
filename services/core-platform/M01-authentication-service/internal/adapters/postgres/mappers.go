package postgres

import (
	"errors"
	"strings"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"gorm.io/gorm"
)

func toDomainUser(row userModel, roleName string) domain.User {
	return domain.User{
		UserID:        row.UserID,
		Email:         row.Email,
		PasswordHash:  row.PasswordHash,
		RoleID:        row.RoleID,
		RoleName:      roleName,
		EmailVerified: row.EmailVerified,
		IsActive:      row.IsActive,
		DeletedAt:     row.DeletedAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func toDomainSession(row sessionModel) domain.Session {
	ip := ""
	if row.IPAddress != nil {
		ip = *row.IPAddress
	}
	return domain.Session{
		SessionID:      row.SessionID,
		UserID:         row.UserID,
		DeviceName:     row.DeviceName,
		DeviceOS:       row.DeviceOS,
		IPAddress:      ip,
		UserAgent:      row.UserAgent,
		CreatedAt:      row.CreatedAt,
		LastActivityAt: row.LastActivityAt,
		ExpiresAt:      row.ExpiresAt,
		RevokedAt:      row.RevokedAt,
	}
}

func toDomainLoginAttempt(row loginAttemptModel) domain.LoginAttempt {
	ip := ""
	if row.IPAddress != nil {
		ip = *row.IPAddress
	}
	return domain.LoginAttempt{
		ID:            row.ID,
		UserID:        row.UserID,
		AttemptAt:     row.AttemptAt,
		IPAddress:     ip,
		Status:        row.Status,
		FailureReason: row.FailureReason,
		DeviceName:    row.DeviceName,
		DeviceOS:      row.DeviceOS,
		UserAgent:     row.UserAgent,
	}
}

func nullableString(v string) *string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isUniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
