package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
)

type Config struct {
	ServiceName                         string
	ProfileCacheTTL                     time.Duration
	UsernameCooldownDays                int
	UsernameRedirectDays                int
	MaxSocialLinks                      int
	IdempotencyTTL                      time.Duration
	EventDedupTTL                       time.Duration
	FeatureProfileCompletenessVisible   bool
	FeatureKYCReverificationInterval    time.Duration
	FeatureFollowerSyncHighEarnerHourly bool
	FeaturePayPalOwnershipVerification  bool
	FeatureAvatarManualRetry            bool
	KYCAnonymizeAfter                   time.Duration
	UsernameHistoryRetention            time.Duration
}

type UpdateProfileRequest struct {
	DisplayName     *string `json:"display_name,omitempty"`
	Bio             *string `json:"bio,omitempty"`
	IsPrivate       *bool   `json:"is_private,omitempty"`
	IsUnlisted      *bool   `json:"is_unlisted,omitempty"`
	HideStatistics  *bool   `json:"hide_statistics,omitempty"`
	AnalyticsOptOut *bool   `json:"analytics_opt_out,omitempty"`
}

type ChangeUsernameRequest struct {
	Username string `json:"username"`
}

type AddSocialLinkRequest struct {
	Platform          string  `json:"platform"`
	ProfileURL        string  `json:"profile_url"`
	OAuthConnectionID *string `json:"oauth_connection_id,omitempty"`
}

type PutPayoutMethodRequest struct {
	MethodType                string `json:"method_type"`
	StripeAccountID           string `json:"stripe_account_id,omitempty"`
	Email                     string `json:"email,omitempty"`
	WalletAddress             string `json:"wallet_address,omitempty"`
	WalletAddressConfirmation string `json:"wallet_address_confirmation,omitempty"`
}

type UploadKYCDocumentRequest struct {
	DocumentType    string
	FileName        string
	FileContentType string
	FileBytes       []byte
}

type AdminKYCDecisionRequest struct {
	UserID          uuid.UUID
	RejectionReason string
	ReviewedBy      uuid.UUID
	Now             time.Time
}

type ProfileResponse struct {
	ProfileID       string             `json:"profile_id,omitempty"`
	UserID          string             `json:"user_id,omitempty"`
	Username        string             `json:"username"`
	DisplayName     string             `json:"display_name"`
	Bio             string             `json:"bio,omitempty"`
	AvatarURL       string             `json:"avatar_url,omitempty"`
	BannerURL       string             `json:"banner_url,omitempty"`
	KYCStatus       string             `json:"kyc_status,omitempty"`
	IsPrivate       bool               `json:"is_private,omitempty"`
	IsUnlisted      bool               `json:"is_unlisted,omitempty"`
	HideStatistics  bool               `json:"hide_statistics,omitempty"`
	AnalyticsOptOut bool               `json:"analytics_opt_out,omitempty"`
	SocialLinks     []SocialLinkView   `json:"social_links,omitempty"`
	PayoutMethods   []PayoutMethodView `json:"payout_methods,omitempty"`
	Statistics      *ProfileStatsView  `json:"statistics,omitempty"`
	Documents       []KYCDocumentView  `json:"documents,omitempty"`
	CreatedAt       time.Time          `json:"created_at,omitempty"`
	UpdatedAt       time.Time          `json:"updated_at,omitempty"`
	MemberSince     time.Time          `json:"member_since,omitempty"`
	Message         string             `json:"message,omitempty"`
}

type SocialLinkView struct {
	Platform   string `json:"platform"`
	Handle     string `json:"handle,omitempty"`
	ProfileURL string `json:"profile_url"`
	Verified   bool   `json:"verified"`
}

type PayoutMethodView struct {
	MethodType         string     `json:"method_type"`
	VerificationStatus string     `json:"verification_status"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
}

type ProfileStatsView struct {
	TotalEarningsYTD float64 `json:"total_earnings_ytd,omitempty"`
	SubmissionCount  int     `json:"submission_count"`
	ApprovalRate     float64 `json:"approval_rate"`
	FollowerCount    int     `json:"follower_count"`
}

type KYCDocumentView struct {
	DocumentType    string     `json:"document_type"`
	Status          string     `json:"status"`
	UploadedAt      time.Time  `json:"uploaded_at"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
}

type UsernameAvailabilityResponse struct {
	Username  string `json:"username"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type AvatarUploadResponse struct {
	UploadID string `json:"upload_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type PublicProfileResponse struct {
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	Bio         string            `json:"bio,omitempty"`
	AvatarURL   string            `json:"avatar_url,omitempty"`
	IsPrivate   bool              `json:"is_private,omitempty"`
	Message     string            `json:"message,omitempty"`
	SocialLinks []SocialLinkView  `json:"social_links,omitempty"`
	Statistics  *ProfileStatsView `json:"statistics,omitempty"`
	MemberSince time.Time         `json:"member_since,omitempty"`
}

func toProfileStatsView(stats domain.ProfileStats) *ProfileStatsView {
	return &ProfileStatsView{
		TotalEarningsYTD: stats.TotalEarningsYTD,
		SubmissionCount:  stats.SubmissionCount,
		ApprovalRate:     stats.ApprovalRate,
		FollowerCount:    stats.FollowerCount,
	}
}
