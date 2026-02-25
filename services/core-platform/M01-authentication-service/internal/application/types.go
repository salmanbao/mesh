package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

// Config is the runtime policy bundle consumed by application service logic.
// Keeping policy centralized avoids drifting defaults across handlers/adapters.
type Config struct {
	DefaultRole                               string
	TokenTTL                                  time.Duration
	SessionTTL                                time.Duration
	SessionAbsoluteTTL                        time.Duration
	FailedLoginThreshold                      int
	LockoutDuration                           time.Duration
	RegisterOIDCFieldMode                     string
	OIDCAllowEmailLinking                     bool
	OIDCAllowedRedirectURIs                   []string
	OIDCCompletionTokenTTL                    time.Duration
	RegisterRateLimitIPThreshold              int
	RegisterRateLimitIdentifierThreshold      int
	RegisterRateLimitWindow                   time.Duration
	OIDCAuthorizeRateLimitIPThreshold         int
	OIDCAuthorizeRateLimitIdentifierThreshold int
	OIDCAuthorizeRateLimitWindow              time.Duration
}

// RegisterRequest is the command payload for local account creation.
// OIDC fields are reserved for staged rollout compatibility.
type RegisterRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Role          string `json:"role"`
	TermsAccepted bool   `json:"terms_accepted"`
	IPAddress     string `json:"-"`

	// Deprecated: OIDC fields are no longer accepted on /auth/v1/register.
	Provider string `json:"provider"`
	// Deprecated: OIDC fields are no longer accepted on /auth/v1/register.
	AuthorizationCode string `json:"authorization_code"`
	// Deprecated: OIDC fields are no longer accepted on /auth/v1/register.
	RedirectURI string `json:"redirect_uri"`
	// Deprecated: OIDC fields are no longer accepted on /auth/v1/register.
	Nonce string `json:"nonce"`
	// Deprecated: OIDC fields are no longer accepted on /auth/v1/register.
	CodeVerifier string `json:"code_verifier"`
}

// RegisterResponse is returned after successful registration.
type RegisterResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token,omitempty"`
	SessionID uuid.UUID `json:"session_id,omitempty"`
	ExpiresIn int64     `json:"expires_in,omitempty"`
}

// LoginRequest captures primary credential authentication context.
type LoginRequest struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	RememberDevice bool   `json:"remember_device"`
	DeviceName     string `json:"device_name"`
	DeviceOS       string `json:"device_os"`
	IPAddress      string `json:"ip_address"`
	UserAgent      string `json:"user_agent"`
}

// LoginResponse carries either a primary token or a temporary 2FA handoff token.
// This dual shape keeps MFA as a first-class step without separate transport contracts.
type LoginResponse struct {
	Requires2FA bool      `json:"requires_2fa"`
	TempToken   string    `json:"temp_token,omitempty"`
	Token       string    `json:"token,omitempty"`
	SessionID   uuid.UUID `json:"session_id,omitempty"`
	ExpiresIn   int64     `json:"expires_in,omitempty"`
}

// TwoFAVerifyRequest validates an MFA challenge and finalizes session issuance.
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

// TwoFASetupRequest modifies enabled second-factor methods for a user.
type TwoFASetupRequest struct {
	Action string `json:"action"`
	Method string `json:"method"`
}

// TwoFASetupResponse returns resulting MFA method state and bootstrap artifacts.
type TwoFASetupResponse struct {
	Method      string   `json:"method"`
	Enabled     bool     `json:"enabled"`
	Secret      string   `json:"secret,omitempty"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

// PasswordResetRequest confirms a reset token and applies a new password.
type PasswordResetRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// OIDCLinkRequest links an external identity to an authenticated user.
type OIDCLinkRequest struct {
	Provider          string `json:"provider"`
	AuthorizationCode string `json:"authorization_code"`
	RedirectURI       string `json:"redirect_uri"`
	Nonce             string `json:"nonce"`
	CodeVerifier      string `json:"code_verifier"`
}

// RegisterCompleteRequest finalizes deferred OIDC onboarding.
type RegisterCompleteRequest struct {
	CompletionToken string `json:"completion_token"`
	Role            string `json:"role,omitempty"`
}

// OIDCAuthorizeResponse returns the generated authorization URL and state handle.
type OIDCAuthorizeResponse struct {
	AuthorizeURL string `json:"authorize_url"`
	State        string `json:"state"`
}

// OIDCCallbackResult is the callback outcome for either login or deferred completion.
type OIDCCallbackResult struct {
	RedirectURL            string    `json:"redirect_url,omitempty"`
	UserID                 uuid.UUID `json:"user_id,omitempty"`
	Token                  string    `json:"token,omitempty"`
	SessionID              uuid.UUID `json:"session_id,omitempty"`
	ExpiresIn              int64     `json:"expires_in,omitempty"`
	RegistrationIncomplete bool      `json:"registration_incomplete,omitempty"`
	CompletionToken        string    `json:"completion_token,omitempty"`
}

// RefreshResponse contains a rotated access token and TTL.
type RefreshResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

// SessionItem is the API projection of a persisted session record.
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

// LoginHistoryQuery controls filtering and paging for login-history retrieval.
type LoginHistoryQuery struct {
	Page   int
	Limit  int
	Days   int
	Status string
}

// LoginHistoryItem is the API projection of a login attempt.
type LoginHistoryItem struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
	FailureReason string    `json:"failure_reason,omitempty"`
	IPAddress     string    `json:"ip_address"`
	DeviceName    string    `json:"device_name,omitempty"`
	DeviceOS      string    `json:"device_os,omitempty"`
}

// UserIdentity is the owner-api projection for downstream services that need
// stable identity fields without direct database access.
type UserIdentity struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Status string    `json:"status"`
}

// toSessionItem maps domain session data into response shape.
// This single mapper keeps field projection consistent across all session endpoints.
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
