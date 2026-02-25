package postgres

import (
	"time"

	"github.com/google/uuid"
)

type profileModel struct {
	ProfileID            uuid.UUID  `gorm:"column:profile_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID               uuid.UUID  `gorm:"column:user_id"`
	Username             string     `gorm:"column:username"`
	DisplayName          string     `gorm:"column:display_name"`
	Bio                  string     `gorm:"column:bio"`
	AvatarURL            string     `gorm:"column:avatar_url"`
	BannerURL            string     `gorm:"column:banner_url"`
	KYCStatus            string     `gorm:"column:kyc_status"`
	IsPrivate            bool       `gorm:"column:is_private"`
	IsUnlisted           bool       `gorm:"column:is_unlisted"`
	HideStatistics       bool       `gorm:"column:hide_statistics"`
	AnalyticsOptOut      bool       `gorm:"column:analytics_opt_out"`
	LastUsernameChangeAt *time.Time `gorm:"column:last_username_change_at"`
	CreatedAt            time.Time  `gorm:"column:created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at"`
	DeletedAt            *time.Time `gorm:"column:deleted_at"`
}

func (profileModel) TableName() string { return "profiles" }

type socialLinkModel struct {
	SocialLinkID      uuid.UUID  `gorm:"column:social_link_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID            uuid.UUID  `gorm:"column:user_id"`
	Platform          string     `gorm:"column:platform"`
	Handle            string     `gorm:"column:handle"`
	ProfileURL        string     `gorm:"column:profile_url"`
	Verified          bool       `gorm:"column:verified"`
	OAuthConnectionID *uuid.UUID `gorm:"column:oauth_connection_id"`
	AddedAt           time.Time  `gorm:"column:added_at"`
	LastSyncedAt      *time.Time `gorm:"column:last_synced_at"`
}

func (socialLinkModel) TableName() string { return "social_links" }

type payoutMethodModel struct {
	PayoutMethodID      uuid.UUID  `gorm:"column:payout_method_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID              uuid.UUID  `gorm:"column:user_id"`
	MethodType          string     `gorm:"column:method_type"`
	IdentifierEncrypted []byte     `gorm:"column:identifier_encrypted"`
	VerificationStatus  string     `gorm:"column:verification_status"`
	AddedAt             time.Time  `gorm:"column:added_at"`
	LastUsedAt          *time.Time `gorm:"column:last_used_at"`
}

func (payoutMethodModel) TableName() string { return "payout_methods" }

type kycDocumentModel struct {
	KYCDocumentID   uuid.UUID  `gorm:"column:kyc_document_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          uuid.UUID  `gorm:"column:user_id"`
	DocumentType    string     `gorm:"column:document_type"`
	FileKey         string     `gorm:"column:file_key"`
	Status          string     `gorm:"column:status"`
	RejectionReason string     `gorm:"column:rejection_reason"`
	UploadedAt      time.Time  `gorm:"column:uploaded_at"`
	ReviewedAt      *time.Time `gorm:"column:reviewed_at"`
	ReviewedBy      *uuid.UUID `gorm:"column:reviewed_by"`
}

func (kycDocumentModel) TableName() string { return "kyc_documents" }

type usernameHistoryModel struct {
	HistoryID         uuid.UUID `gorm:"column:history_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID            uuid.UUID `gorm:"column:user_id"`
	OldUsername       string    `gorm:"column:old_username"`
	NewUsername       string    `gorm:"column:new_username"`
	ChangedAt         time.Time `gorm:"column:changed_at"`
	RedirectExpiresAt time.Time `gorm:"column:redirect_expires_at"`
}

func (usernameHistoryModel) TableName() string { return "username_history" }

type profileStatsModel struct {
	StatID           uuid.UUID `gorm:"column:stat_id;type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID `gorm:"column:user_id"`
	TotalEarningsYTD float64   `gorm:"column:total_earnings_ytd"`
	SubmissionCount  int       `gorm:"column:submission_count"`
	ApprovalRate     float64   `gorm:"column:approval_rate"`
	FollowerCount    int       `gorm:"column:follower_count"`
	LastUpdatedAt    time.Time `gorm:"column:last_updated_at"`
}

func (profileStatsModel) TableName() string { return "profile_stats" }

type reservedUsernameModel struct {
	ReservedUsernameID uuid.UUID `gorm:"column:reserved_username_id;type:uuid;default:gen_random_uuid();primaryKey"`
	Username           string    `gorm:"column:username"`
	ReservedAt         time.Time `gorm:"column:reserved_at"`
	Reason             string    `gorm:"column:reason"`
}

func (reservedUsernameModel) TableName() string { return "reserved_usernames" }

type profileOutboxModel struct {
	OutboxID         uuid.UUID  `gorm:"column:outbox_id;type:uuid;primaryKey"`
	EventType        string     `gorm:"column:event_type"`
	PartitionKey     string     `gorm:"column:partition_key"`
	PartitionKeyPath string     `gorm:"column:partition_key_path"`
	Payload          string     `gorm:"column:payload"`
	SchemaVersion    string     `gorm:"column:schema_version"`
	TraceID          string     `gorm:"column:trace_id"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	FirstSeenAt      time.Time  `gorm:"column:first_seen_at"`
	PublishedAt      *time.Time `gorm:"column:published_at"`
	RetryCount       int        `gorm:"column:retry_count"`
	LastError        *string    `gorm:"column:last_error"`
	LastErrorAt      *time.Time `gorm:"column:last_error_at"`
}

func (profileOutboxModel) TableName() string { return "profile_outbox" }

type profileIdempotencyModel struct {
	IdempotencyKey string    `gorm:"column:idempotency_key;primaryKey"`
	RequestHash    string    `gorm:"column:request_hash"`
	Status         string    `gorm:"column:status"`
	ResponseCode   int       `gorm:"column:response_code"`
	ResponseBody   *string   `gorm:"column:response_body"`
	ExpiresAt      time.Time `gorm:"column:expires_at"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (profileIdempotencyModel) TableName() string { return "profile_idempotency" }

type profileEventDedupModel struct {
	EventID     string    `gorm:"column:event_id;primaryKey"`
	EventType   string    `gorm:"column:event_type"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
}

func (profileEventDedupModel) TableName() string { return "profile_event_dedup" }
