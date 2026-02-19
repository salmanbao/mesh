package domain

import (
	"time"

	"github.com/google/uuid"
)

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
