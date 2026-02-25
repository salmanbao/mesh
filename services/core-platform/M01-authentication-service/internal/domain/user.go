package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is the canonical authentication identity aggregate for M01.
// It keeps only auth-relevant state so authorization/session flows stay service-owned.
type User struct {
	UserID        uuid.UUID
	Email         string
	PasswordHash  string
	RoleID        uuid.UUID
	RoleName      string
	EmailVerified bool
	IsActive      bool
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Session models a login session issued by M01.
// We persist this separately to support per-device revocation and session history.
type Session struct {
	SessionID      uuid.UUID
	UserID         uuid.UUID
	DeviceName     string
	DeviceOS       string
	IPAddress      string
	UserAgent      string
	CreatedAt      time.Time
	LastActivityAt time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
}

// LoginAttempt records authentication outcomes for audit and lockout controls.
// The reason for this explicit model is to keep fraud/risk signal generation deterministic.
type LoginAttempt struct {
	ID            int64
	UserID        *uuid.UUID
	AttemptAt     time.Time
	IPAddress     string
	Status        string
	FailureReason string
	DeviceName    string
	DeviceOS      string
	UserAgent     string
}
