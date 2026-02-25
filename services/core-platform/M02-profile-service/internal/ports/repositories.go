package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
)

type CreateProfileParams struct {
	UserID      uuid.UUID
	Username    string
	DisplayName string
	CreatedAt   time.Time
}

type UpdateProfileParams struct {
	UserID          uuid.UUID
	DisplayName     *string
	Bio             *string
	IsPrivate       *bool
	IsUnlisted      *bool
	HideStatistics  *bool
	AnalyticsOptOut *bool
	AvatarURL       *string
	BannerURL       *string
	UpdatedAt       time.Time
}

type CreateSocialLinkParams struct {
	UserID            uuid.UUID
	Platform          string
	Handle            string
	ProfileURL        string
	Verified          bool
	OAuthConnectionID *uuid.UUID
	AddedAt           time.Time
}

type PutPayoutMethodParams struct {
	UserID              uuid.UUID
	MethodType          string
	IdentifierEncrypted []byte
	VerificationStatus  string
	Now                 time.Time
}

type CreateKYCDocumentParams struct {
	UserID       uuid.UUID
	DocumentType string
	FileKey      string
	Status       string
	UploadedAt   time.Time
}

type UpsertProfileStatsParams struct {
	UserID           uuid.UUID
	TotalEarningsYTD float64
	SubmissionCount  int
	ApprovalRate     float64
	FollowerCount    int
	UpdatedAt        time.Time
}

type UsernameAvailability struct {
	Available bool
	Reason    string
}

type ProfileRepository interface {
	CreateProfileWithDefaults(ctx context.Context, params CreateProfileParams) (domain.Profile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (domain.Profile, error)
	GetByUsername(ctx context.Context, username string) (domain.Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (domain.Profile, error)
	UpdateUsername(ctx context.Context, userID uuid.UUID, newUsername string, now time.Time, redirectDays int) (oldUsername string, updated domain.Profile, err error)
	CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailability, error)
	SoftDeleteByUserID(ctx context.Context, userID uuid.UUID, deletedAt time.Time) error
}

type SocialLinkRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.SocialLink, error)
	Create(ctx context.Context, params CreateSocialLinkParams) (domain.SocialLink, error)
	DeleteByUserAndPlatform(ctx context.Context, userID uuid.UUID, platform string) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type PayoutMethodRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.PayoutMethod, error)
	Upsert(ctx context.Context, params PutPayoutMethodParams) (domain.PayoutMethod, error)
}

type KYCRepository interface {
	CreateDocument(ctx context.Context, params CreateKYCDocumentParams) (domain.KYCDocument, error)
	ListDocumentsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.KYCDocument, error)
	UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, rejectionReason string, reviewedAt time.Time, reviewedBy *uuid.UUID) error
	ListPendingQueue(ctx context.Context, limit, offset int) ([]domain.Profile, error)
}

type ProfileStatsRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (domain.ProfileStats, error)
	Upsert(ctx context.Context, params UpsertProfileStatsParams) error
}

type ReservedUsernameRepository interface {
	IsReserved(ctx context.Context, username string) (bool, error)
}

type UsernameHistoryRepository interface {
	ResolveRedirect(ctx context.Context, oldUsername string, now time.Time) (newUsername string, found bool, err error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]domain.UsernameHistory, error)
}

type ProfileReadModel struct {
	Profile       domain.Profile
	SocialLinks   []domain.SocialLink
	PayoutMethods []domain.PayoutMethod
	Stats         domain.ProfileStats
	Documents     []domain.KYCDocument
}

type ReadRepository interface {
	GetProfileReadModelByUserID(ctx context.Context, userID uuid.UUID) (ProfileReadModel, error)
	GetPublicProfileByUsername(ctx context.Context, username string, now time.Time) (ProfileReadModel, bool, error)
}

type OutboxEvent struct {
	EventID          uuid.UUID
	EventType        string
	PartitionKey     string
	PartitionKeyPath string
	Payload          []byte
	OccurredAt       time.Time
	SchemaVersion    string
	TraceID          string
}

type OutboxRecord struct {
	OutboxID     uuid.UUID
	EventType    string
	PartitionKey string
	Payload      []byte
	RetryCount   int
	PublishedAt  *time.Time
	LastError    *string
	LastErrorAt  *time.Time
	FirstSeenAt  time.Time
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, event OutboxEvent) error
	FetchUnpublished(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkPublished(ctx context.Context, outboxID uuid.UUID, at time.Time) error
	MarkFailed(ctx context.Context, outboxID uuid.UUID, errMsg string, at time.Time) error
}

type EventDedupRepository interface {
	IsDuplicate(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string, expiresAt time.Time) error
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	Status       string
	ResponseCode int
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseCode int, responseBody []byte, at time.Time) error
}
