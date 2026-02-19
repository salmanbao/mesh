package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

type Config struct {
	DefaultRole          string
	TokenTTL             time.Duration
	SessionTTL           time.Duration
	SessionAbsoluteTTL   time.Duration
	FailedLoginThreshold int
	LockoutDuration      time.Duration
}

type RegisterRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Role          string `json:"role"`
	TermsAccepted bool   `json:"terms_accepted"`

	Provider          string `json:"provider"`
	AuthorizationCode string `json:"authorization_code"`
}

type RegisterResponse struct {
	UserID uuid.UUID `json:"user_id"`
}

type LoginRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	RememberDevice bool   `json:"remember_device"`
	DeviceName     string `json:"device_name"`
	DeviceOS       string `json:"device_os"`
	IPAddress      string `json:"ip_address"`
	UserAgent      string `json:"user_agent"`
}

type LoginResponse struct {
	Requires2FA bool      `json:"requires_2fa"`
	TempToken   string    `json:"temp_token,omitempty"`
	Token       string    `json:"token,omitempty"`
	SessionID   uuid.UUID `json:"session_id,omitempty"`
	ExpiresIn   int64     `json:"expires_in,omitempty"`
}

type TwoFAVerifyRequest struct {
	TempToken      string `json:"temp_token"`
	Code           string `json:"code"`
	Method         string `json:"method"`
	RememberDevice bool   `json:"remember_device"`
	DeviceName     string `json:"device_name"`
	DeviceOS       string `json:"device_os"`
	IPAddress      string `json:"ip_address"`
	UserAgent      string `json:"user_agent"`
}

type TwoFASetupRequest struct {
	Action string `json:"action"`
	Method string `json:"method"`
}

type TwoFASetupResponse struct {
	Method      string   `json:"method"`
	Enabled     bool     `json:"enabled"`
	Secret      string   `json:"secret,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

type PasswordResetRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type OIDCLinkRequest struct {
	Provider          string `json:"provider"`
	AuthorizationCode string `json:"authorization_code"`
	RedirectURI       string `json:"redirect_uri"`
	Nonce             string `json:"nonce"`
	CodeVerifier      string `json:"code_verifier"`
}

type RefreshResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

type SessionItem struct {
	SessionID      uuid.UUID  `json:"session_id"`
	DeviceName     string     `json:"device_name"`
	DeviceOS       string     `json:"device_os"`
	IPAddress      string     `json:"ip_address"`
	CreatedAt      time.Time  `json:"created_at"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	IsCurrent      bool       `json:"is_current"`
}

type LoginHistoryQuery struct {
	Page   int
	Limit  int
	Days   int
	Status string
}

type LoginHistoryItem struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
	FailureReason string    `json:"failure_reason,omitempty"`
	IPAddress     string    `json:"ip_address"`
	DeviceName    string    `json:"device_name,omitempty"`
	DeviceOS      string    `json:"device_os,omitempty"`
}

func toSessionItem(s domain.Session, currentSessionID uuid.UUID) SessionItem {
	return SessionItem{
		SessionID:      s.SessionID,
		DeviceName:     s.DeviceName,
		DeviceOS:       s.DeviceOS,
		IPAddress:      s.IPAddress,
		CreatedAt:      s.CreatedAt,
		LastActivityAt: s.LastActivityAt,
		ExpiresAt:      s.ExpiresAt,
		RevokedAt:      s.RevokedAt,
		IsCurrent:      s.SessionID == currentSessionID,
	}
}
