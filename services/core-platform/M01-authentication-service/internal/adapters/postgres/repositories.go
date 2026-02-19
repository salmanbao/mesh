package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repositories struct {
	Users         ports.UserRepository
	Sessions      ports.SessionRepository
	LoginAttempts ports.LoginAttemptRepository
	Outbox        ports.OutboxRepository
	Idempotency   ports.IdempotencyRepository
	Recovery      ports.RecoveryRepository
	Credentials   ports.CredentialRepository
	MFA           ports.MFARepository
	OIDC          ports.OIDCRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{
		Users:         &userRepository{db: db},
		Sessions:      &sessionRepository{db: db},
		LoginAttempts: &loginAttemptRepository{db: db},
		Outbox:        &outboxRepository{db: db},
		Idempotency:   &idempotencyRepository{db: db},
		Recovery:      &recoveryRepository{db: db},
		Credentials:   &credentialRepository{db: db},
		MFA:           &mfaRepository{db: db},
		OIDC:          &oidcRepository{db: db},
	}
}

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
	Provider          string    `gorm:"column:provider"`
	ProviderUserID    string    `gorm:"column:provider_user_id"`
	Email             string    `gorm:"column:email"`
	LinkedAt          time.Time `gorm:"column:linked_at"`
	IsPrimary         bool      `gorm:"column:is_primary"`
	Status            string    `gorm:"column:status"`
}

func (oauthConnectionModel) TableName() string { return "oauth_connections" }

type userRepository struct {
	db *gorm.DB
}

func (r *userRepository) CreateWithOutboxTx(ctx context.Context, params ports.CreateUserTxParams, outboxEvent ports.OutboxEvent) (domain.User, error) {
	var result domain.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role roleModel
		if err := tx.Where("name = ?", params.RoleName).Take(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrRoleResolutionFailed
			}
			return err
		}

		rec := userModel{
			Email:         params.Email,
			PasswordHash:  params.PasswordHash,
			RoleID:        role.RoleID,
			EmailVerified: params.EmailVerified,
			CreatedAt:     params.RegisteredAtUTC,
			UpdatedAt:     params.RegisteredAtUTC,
		}
		if err := tx.Create(&rec).Error; err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return err
		}

		payload := outboxEvent.Payload
		if len(payload) == 0 {
			payload = []byte(`{}`)
		}
		var payloadObj map[string]any
		if err := json.Unmarshal(payload, &payloadObj); err == nil {
			payloadObj["user_id"] = rec.UserID.String()
			if adjusted, mErr := json.Marshal(payloadObj); mErr == nil {
				payload = adjusted
			}
		}

		outbox := authOutboxModel{
			OutboxID:     outboxEvent.EventID,
			EventType:    outboxEvent.EventType,
			PartitionKey: rec.UserID.String(),
			Payload:      string(payload),
			CreatedAt:    outboxEvent.OccurredAt,
			FirstSeenAt:  outboxEvent.OccurredAt,
		}
		if err := tx.Create(&outbox).Error; err != nil {
			return err
		}

		result = toDomainUser(rec, role.Name)
		return nil
	})
	if err != nil {
		return domain.User{}, err
	}
	return result, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	var rec userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	roleName, err := r.loadRoleName(ctx, rec.RoleID)
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(rec, roleName), nil
}

func (r *userRepository) GetByID(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	var rec userModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	roleName, err := r.loadRoleName(ctx, rec.RoleID)
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(rec, roleName), nil
}

func (r *userRepository) loadRoleName(ctx context.Context, roleID uuid.UUID) (string, error) {
	var role roleModel
	if err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Take(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ErrRoleResolutionFailed
		}
		return "", err
	}
	return role.Name, nil
}

type sessionRepository struct {
	db *gorm.DB
}

func (r *sessionRepository) Create(ctx context.Context, params ports.SessionCreateParams) (domain.Session, error) {
	rec := sessionModel{
		UserID:         params.UserID,
		DeviceName:     params.DeviceName,
		DeviceOS:       params.DeviceOS,
		IPAddress:      nullableString(params.IPAddress),
		UserAgent:      params.UserAgent,
		CreatedAt:      params.LastActivityAt,
		LastActivityAt: params.LastActivityAt,
		ExpiresAt:      params.ExpiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return domain.Session{}, err
	}
	return toDomainSession(rec), nil
}

func (r *sessionRepository) GetByID(ctx context.Context, sessionID uuid.UUID) (domain.Session, error) {
	var rec sessionModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Session{}, domain.ErrNotFound
		}
		return domain.Session{}, err
	}
	return toDomainSession(rec), nil
}

func (r *sessionRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Session, error) {
	var rows []sessionModel
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Session, 0, len(rows))
	for _, item := range rows {
		result = append(result, toDomainSession(item))
	}
	return result, nil
}

func (r *sessionRepository) TouchActivity(ctx context.Context, sessionID uuid.UUID, touchedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("session_id = ?", sessionID).
		Update("last_activity_at", touchedAt).Error
}

func (r *sessionRepository) RevokeByID(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("session_id = ?", sessionID).
		Where("revoked_at IS NULL").
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var exists int64
		if err := r.db.WithContext(ctx).Model(&sessionModel{}).Where("session_id = ?", sessionID).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return domain.ErrNotFound
		}
	}
	return nil
}

func (r *sessionRepository) RevokeAllByUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Update("revoked_at", revokedAt).Error
}

type loginAttemptRepository struct {
	db *gorm.DB
}

func (r *loginAttemptRepository) Insert(ctx context.Context, attempt domain.LoginAttempt) error {
	rec := loginAttemptModel{
		UserID:        attempt.UserID,
		AttemptAt:     attempt.AttemptAt,
		IPAddress:     nullableString(attempt.IPAddress),
		Status:        attempt.Status,
		FailureReason: attempt.FailureReason,
		DeviceName:    attempt.DeviceName,
		DeviceOS:      attempt.DeviceOS,
		UserAgent:     attempt.UserAgent,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *loginAttemptRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int, since *time.Time, status string) ([]domain.LoginAttempt, error) {
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID)
	if since != nil {
		query = query.Where("attempt_at >= ?", *since)
	}
	status = strings.TrimSpace(status)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var rows []loginAttemptModel
	if err := query.Order("attempt_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.LoginAttempt, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainLoginAttempt(row))
	}
	return result, nil
}

type outboxRepository struct {
	db *gorm.DB
}

func (r *outboxRepository) Enqueue(ctx context.Context, event ports.OutboxEvent) error {
	rec := authOutboxModel{
		OutboxID:     event.EventID,
		EventType:    event.EventType,
		PartitionKey: event.PartitionKey,
		Payload:      string(event.Payload),
		CreatedAt:    event.OccurredAt,
		FirstSeenAt:  event.OccurredAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *outboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]ports.OutboxRecord, error) {
	var rows []authOutboxModel
	if err := r.db.WithContext(ctx).
		Where("published_at IS NULL").
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]ports.OutboxRecord, 0, len(rows))
	for _, row := range rows {
		item := ports.OutboxRecord{
			OutboxID:     row.OutboxID,
			EventType:    row.EventType,
			PartitionKey: row.PartitionKey,
			Payload:      []byte(row.Payload),
			RetryCount:   row.RetryCount,
			LastError:    row.LastError,
			CreatedAt:    row.CreatedAt,
			PublishedAt:  row.PublishedAt,
			LastErrorAt:  row.LastErrorAt,
			FirstSeenAt:  row.FirstSeenAt,
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *outboxRepository) MarkPublished(ctx context.Context, outboxID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authOutboxModel{}).
		Where("outbox_id = ?", outboxID).
		Update("published_at", at).Error
}

func (r *outboxRepository) MarkFailed(ctx context.Context, outboxID uuid.UUID, errMsg string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&authOutboxModel{}).
		Where("outbox_id = ?", outboxID).
		Updates(map[string]any{
			"retry_count":   gorm.Expr("retry_count + 1"),
			"last_error":    errMsg,
			"last_error_at": at,
		}).Error
}

type idempotencyRepository struct {
	db *gorm.DB
}

func (r *idempotencyRepository) Get(ctx context.Context, key string) (*ports.IdempotencyRecord, error) {
	var rec authIdempotencyModel
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := ports.IdempotencyRecord{
		Key:          rec.IdempotencyKey,
		RequestHash:  rec.RequestHash,
		Status:       rec.Status,
		ResponseCode: rec.ResponseCode,
		ExpiresAt:    rec.ExpiresAt,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
	if rec.ResponseBody != nil {
		out.ResponseBody = []byte(*rec.ResponseBody)
	}
	return &out, nil
}

func (r *idempotencyRepository) Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error {
	rec := authIdempotencyModel{
		IdempotencyKey: key,
		RequestHash:    requestHash,
		Status:         "PENDING",
		ExpiresAt:      expiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (r *idempotencyRepository) Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error {
	var body *string
	if len(responseBody) > 0 {
		raw := string(responseBody)
		body = &raw
	}
	return r.db.WithContext(ctx).
		Model(&authIdempotencyModel{}).
		Where("idempotency_key = ?", key).
		Updates(map[string]any{
			"status":        "COMPLETED",
			"response_code": responseCode,
			"response_body": body,
			"updated_at":    at,
		}).Error
}

type recoveryRepository struct {
	db *gorm.DB
}

func (r *recoveryRepository) CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error {
	rec := passwordResetTokenModel{
		UserID:    userID,
		TokenHash: tokenHash,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *recoveryRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, usedAt time.Time) (uuid.UUID, error) {
	var rec passwordResetTokenModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", tokenHash).
			Where("used_at IS NULL").
			Where("expires_at > ?", usedAt).
			Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		return tx.Model(&passwordResetTokenModel{}).
			Where("token_id = ?", rec.TokenID).
			Update("used_at", usedAt).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return rec.UserID, nil
}

func (r *recoveryRepository) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, createdAt, expiresAt time.Time) error {
	rec := emailVerificationTokenModel{
		UserID:    userID,
		TokenHash: tokenHash,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *recoveryRepository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string, verifiedAt time.Time) (uuid.UUID, error) {
	var rec emailVerificationTokenModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", tokenHash).
			Where("verified_at IS NULL").
			Where("expires_at > ?", verifiedAt).
			Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		return tx.Model(&emailVerificationTokenModel{}).
			Where("token_id = ?", rec.TokenID).
			Update("verified_at", verifiedAt).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return rec.UserID, nil
}

type credentialRepository struct {
	db *gorm.DB
}

func (r *credentialRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, updatedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"updated_at":    updatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *credentialRepository) SetEmailVerified(ctx context.Context, userID uuid.UUID, verified bool, updatedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&userModel{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"email_verified": verified,
			"updated_at":     updatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *credentialRepository) HasPassword(ctx context.Context, userID uuid.UUID) (bool, error) {
	var rec struct {
		PasswordHash *string `gorm:"column:password_hash"`
	}
	if err := r.db.WithContext(ctx).
		Model(&userModel{}).
		Select("password_hash").
		Where("user_id = ?", userID).
		Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, domain.ErrNotFound
		}
		return false, err
	}
	return rec.PasswordHash != nil && strings.TrimSpace(*rec.PasswordHash) != "", nil
}

type mfaRepository struct {
	db *gorm.DB
}

func (r *mfaRepository) ListEnabledMethods(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var methods []string
	if err := r.db.WithContext(ctx).
		Model(&twoFactorMethodModel{}).
		Where("user_id = ?", userID).
		Where("is_enabled = TRUE").
		Order("is_primary DESC, method_type ASC").
		Pluck("method_type", &methods).Error; err != nil {
		return nil, err
	}
	return methods, nil
}

func (r *mfaRepository) SetMethodEnabled(ctx context.Context, userID uuid.UUID, method string, enabled bool, isPrimary bool, updatedAt time.Time) error {
	rec := twoFactorMethodModel{
		UserID:     userID,
		MethodType: method,
		IsEnabled:  enabled,
		IsPrimary:  isPrimary,
		CreatedAt:  updatedAt,
		UpdatedAt:  updatedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "method_type"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"is_enabled": rec.IsEnabled,
			"is_primary": rec.IsPrimary,
			"updated_at": rec.UpdatedAt,
		}),
	}).Create(&rec).Error
}

func (r *mfaRepository) UpsertTOTPSecret(ctx context.Context, userID uuid.UUID, secretEncrypted []byte, updatedAt time.Time) error {
	rec := totpSecretModel{
		UserID:          userID,
		SecretEncrypted: secretEncrypted,
		CreatedAt:       updatedAt,
		ActivatedAt:     &updatedAt,
		DeactivatedAt:   nil,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"secret_encrypted": rec.SecretEncrypted,
			"activated_at":     rec.ActivatedAt,
			"deactivated_at":   nil,
		}),
	}).Create(&rec).Error
}

func (r *mfaRepository) ReplaceBackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string, createdAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&backupCodeModel{}).Error; err != nil {
			return err
		}
		if len(codeHashes) == 0 {
			return nil
		}
		records := make([]backupCodeModel, 0, len(codeHashes))
		for _, hash := range codeHashes {
			records = append(records, backupCodeModel{
				UserID:    userID,
				CodeHash:  hash,
				CreatedAt: createdAt,
			})
		}
		return tx.Create(&records).Error
	})
}

func (r *mfaRepository) ConsumeBackupCode(ctx context.Context, userID uuid.UUID, codeHash string, usedAt time.Time) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&backupCodeModel{}).
		Where("user_id = ?", userID).
		Where("code_hash = ?", codeHash).
		Where("used_at IS NULL").
		Update("used_at", usedAt)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

type oidcRepository struct {
	db *gorm.DB
}

func (r *oidcRepository) FindUserByProviderSubject(ctx context.Context, provider, providerUserID string) (uuid.UUID, error) {
	var rec oauthConnectionModel
	if err := r.db.WithContext(ctx).
		Where("provider = ?", provider).
		Where("provider_user_id = ?", providerUserID).
		Where("status = 'ACTIVE'").
		Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, domain.ErrNotFound
		}
		return uuid.Nil, err
	}
	return rec.UserID, nil
}

func (r *oidcRepository) UpsertConnection(ctx context.Context, userID uuid.UUID, provider, providerUserID, email string, isPrimary bool, now time.Time) error {
	rec := oauthConnectionModel{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		LinkedAt:       now,
		IsPrimary:      isPrimary,
		Status:         "ACTIVE",
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "provider"},
			{Name: "provider_user_id"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"user_id":    rec.UserID,
			"email":      rec.Email,
			"is_primary": rec.IsPrimary,
			"status":     "ACTIVE",
			"linked_at":  rec.LinkedAt,
		}),
	}).Create(&rec).Error
}

func (r *oidcRepository) CountConnections(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&oauthConnectionModel{}).
		Where("user_id = ?", userID).
		Where("status = 'ACTIVE'").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *oidcRepository) DeleteConnection(ctx context.Context, userID uuid.UUID, provider string) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&oauthConnectionModel{}).
		Where("user_id = ?", userID).
		Where("provider = ?", provider).
		Where("status = 'ACTIVE'").
		Update("status", "REVOKED")
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

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
