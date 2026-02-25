package domain

import (
	"time"

	"github.com/google/uuid"
)

type KYCStatus string

const (
	KYCStatusNotStarted         KYCStatus = "not_started"
	KYCStatusPending            KYCStatus = "pending"
	KYCStatusVerified           KYCStatus = "verified"
	KYCStatusRejected           KYCStatus = "rejected"
	KYCStatusUnderInvestigation KYCStatus = "under_investigation"
)

type Profile struct {
	ProfileID            uuid.UUID
	UserID               uuid.UUID
	Username             string
	DisplayName          string
	Bio                  string
	AvatarURL            string
	BannerURL            string
	KYCStatus            KYCStatus
	IsPrivate            bool
	IsUnlisted           bool
	HideStatistics       bool
	AnalyticsOptOut      bool
	LastUsernameChangeAt *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

type SocialLink struct {
	SocialLinkID      uuid.UUID
	UserID            uuid.UUID
	Platform          string
	Handle            string
	ProfileURL        string
	Verified          bool
	OAuthConnectionID *uuid.UUID
	AddedAt           time.Time
	LastSyncedAt      *time.Time
}

type PayoutMethod struct {
	PayoutMethodID      uuid.UUID
	UserID              uuid.UUID
	MethodType          string
	IdentifierEncrypted []byte
	VerificationStatus  string
	AddedAt             time.Time
	LastUsedAt          *time.Time
}

type KYCDocument struct {
	KYCDocumentID   uuid.UUID
	UserID          uuid.UUID
	DocumentType    string
	FileKey         string
	Status          string
	RejectionReason string
	UploadedAt      time.Time
	ReviewedAt      *time.Time
	ReviewedBy      *uuid.UUID
}

type UsernameHistory struct {
	HistoryID         uuid.UUID
	UserID            uuid.UUID
	OldUsername       string
	NewUsername       string
	ChangedAt         time.Time
	RedirectExpiresAt time.Time
}

type ProfileStats struct {
	StatID           uuid.UUID
	UserID           uuid.UUID
	TotalEarningsYTD float64
	SubmissionCount  int
	ApprovalRate     float64
	FollowerCount    int
	LastUpdatedAt    time.Time
}

type UserIdentity struct {
	UserID uuid.UUID
	Email  string
	Role   string
	Status string
}
