package postgres

import (
	"time"

	"github.com/google/uuid"
)

type roleModel struct {
	RoleID    uuid.UUID `gorm:"column:role_id;type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (roleModel) TableName() string { return "roles" }

type userModel struct {
	UserID        uuid.UUID  `gorm:"column:user_id;type:uuid;default:gen_random_uuid();primaryKey"`
	Email         string     `gorm:"column:email"`
	PasswordHash  string     `gorm:"column:password_hash"`
	RoleID        uuid.UUID  `gorm:"column:role_id"`
	EmailVerified bool       `gorm:"column:email_verified"`
	IsActive      bool       `gorm:"column:is_active"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

func (userModel) TableName() string { return "users" }

type sessionModel struct {
	SessionID      uuid.UUID  `gorm:"column:session_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         uuid.UUID  `gorm:"column:user_id"`
	DeviceName     string     `gorm:"column:device_name"`
	DeviceOS       string     `gorm:"column:device_os"`
	IPAddress      *string    `gorm:"column:ip_address"`
	UserAgent      string     `gorm:"column:user_agent"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	LastActivityAt time.Time  `gorm:"column:last_activity_at"`
	ExpiresAt      time.Time  `gorm:"column:expires_at"`
	RevokedAt      *time.Time `gorm:"column:revoked_at"`
}

func (sessionModel) TableName() string { return "sessions" }

type loginAttemptModel struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	UserID        *uuid.UUID `gorm:"column:user_id"`
	AttemptAt     time.Time  `gorm:"column:attempt_at"`
	IPAddress     *string    `gorm:"column:ip_address"`
	Status        string     `gorm:"column:status"`
	FailureReason string     `gorm:"column:failure_reason"`
	DeviceName    string     `gorm:"column:device_name"`
	DeviceOS      string     `gorm:"column:device_os"`
	UserAgent     string     `gorm:"column:user_agent"`
}

func (loginAttemptModel) TableName() string { return "login_attempts" }

type authOutboxModel struct {
	OutboxID     uuid.UUID  `gorm:"column:outbox_id;type:uuid;primaryKey"`
	EventType    string     `gorm:"column:event_type"`
	PartitionKey string     `gorm:"column:partition_key"`
	Payload      string     `gorm:"column:payload;type:jsonb"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	FirstSeenAt  time.Time  `gorm:"column:first_seen_at"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
	RetryCount   int        `gorm:"column:retry_count"`
	LastError    *string    `gorm:"column:last_error"`
	LastErrorAt  *time.Time `gorm:"column:last_error_at"`
	ClaimToken   *string    `gorm:"column:claim_token"`
	ClaimUntil   *time.Time `gorm:"column:claim_until"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
}

func (authOutboxModel) TableName() string { return "auth_outbox" }

type authIdempotencyModel struct {
	IdempotencyKey string    `gorm:"column:idempotency_key;primaryKey"`
	RequestHash    string    `gorm:"column:request_hash"`
	Status         string    `gorm:"column:status"`
	ResponseCode   int       `gorm:"column:response_code"`
	ResponseBody   *string   `gorm:"column:response_body;type:jsonb"`
	ExpiresAt      time.Time `gorm:"column:expires_at"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (authIdempotencyModel) TableName() string { return "auth_idempotency" }

type passwordResetTokenModel struct {
	TokenID   uuid.UUID  `gorm:"column:token_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID  `gorm:"column:user_id"`
	TokenHash string     `gorm:"column:token_hash"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
}

func (passwordResetTokenModel) TableName() string { return "password_reset_tokens" }

type emailVerificationTokenModel struct {
	TokenID    uuid.UUID  `gorm:"column:token_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id"`
	TokenHash  string     `gorm:"column:token_hash"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	VerifiedAt *time.Time `gorm:"column:verified_at"`
}

func (emailVerificationTokenModel) TableName() string { return "email_verification_tokens" }

type twoFactorMethodModel struct {
	MethodID   uuid.UUID `gorm:"column:method_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID `gorm:"column:user_id"`
	MethodType string    `gorm:"column:method_type"`
	IsEnabled  bool      `gorm:"column:is_enabled"`
	IsPrimary  bool      `gorm:"column:is_primary"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (twoFactorMethodModel) TableName() string { return "two_factor_methods" }

type totpSecretModel struct {
	SecretID        uuid.UUID  `gorm:"column:secret_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          uuid.UUID  `gorm:"column:user_id"`
	SecretEncrypted []byte     `gorm:"column:secret_encrypted"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	ActivatedAt     *time.Time `gorm:"column:activated_at"`
	DeactivatedAt   *time.Time `gorm:"column:deactivated_at"`
}

func (totpSecretModel) TableName() string { return "totp_secrets" }

type backupCodeModel struct {
	BackupCodeID uuid.UUID  `gorm:"column:backup_code_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       uuid.UUID  `gorm:"column:user_id"`
	CodeHash     string     `gorm:"column:code_hash"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UsedAt       *time.Time `gorm:"column:used_at"`
}

func (backupCodeModel) TableName() string { return "backup_codes" }

type oauthConnectionModel struct {
	OAuthConnectionID uuid.UUID `gorm:"column:oauth_connection_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID            uuid.UUID `gorm:"column:user_id"`
	Issuer            string    `gorm:"column:issuer"`
	Subject           string    `gorm:"column:subject"`
	Provider          string    `gorm:"column:provider"`
	ProviderUserID    string    `gorm:"column:provider_user_id"`
	Email             string    `gorm:"column:email"`
	EmailAtLinkTime   string    `gorm:"column:email_at_link_time"`
	LinkedAt          time.Time `gorm:"column:linked_at"`
	LastLoginAt       time.Time `gorm:"column:last_login_at"`
	IsPrimary         bool      `gorm:"column:is_primary"`
	Status            string    `gorm:"column:status"`
}

func (oauthConnectionModel) TableName() string { return "oauth_connections" }

type oauthTokenModel struct {
	OAuthTokenID  uuid.UUID  `gorm:"column:oauth_token_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        uuid.UUID  `gorm:"column:user_id"`
	Provider      string     `gorm:"column:provider"`
	AccessToken   string     `gorm:"column:access_token"`
	RefreshToken  *string    `gorm:"column:refresh_token"`
	ExpiresAt     *time.Time `gorm:"column:expires_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (oauthTokenModel) TableName() string { return "oauth_tokens" }
