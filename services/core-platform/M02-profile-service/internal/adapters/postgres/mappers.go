package postgres

import "github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"

func toDomainProfile(m profileModel) domain.Profile {
	return domain.Profile{
		ProfileID: m.ProfileID, UserID: m.UserID, Username: m.Username, DisplayName: m.DisplayName,
		Bio: m.Bio, AvatarURL: m.AvatarURL, BannerURL: m.BannerURL, KYCStatus: domain.KYCStatus(m.KYCStatus),
		IsPrivate: m.IsPrivate, IsUnlisted: m.IsUnlisted, HideStatistics: m.HideStatistics,
		AnalyticsOptOut: m.AnalyticsOptOut, LastUsernameChangeAt: m.LastUsernameChangeAt,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, DeletedAt: m.DeletedAt,
	}
}

func toDomainSocialLink(m socialLinkModel) domain.SocialLink {
	return domain.SocialLink{
		SocialLinkID: m.SocialLinkID, UserID: m.UserID, Platform: m.Platform, Handle: m.Handle,
		ProfileURL: m.ProfileURL, Verified: m.Verified, OAuthConnectionID: m.OAuthConnectionID,
		AddedAt: m.AddedAt, LastSyncedAt: m.LastSyncedAt,
	}
}

func toDomainPayoutMethod(m payoutMethodModel) domain.PayoutMethod {
	return domain.PayoutMethod{
		PayoutMethodID: m.PayoutMethodID, UserID: m.UserID, MethodType: m.MethodType,
		IdentifierEncrypted: m.IdentifierEncrypted, VerificationStatus: m.VerificationStatus,
		AddedAt: m.AddedAt, LastUsedAt: m.LastUsedAt,
	}
}

func toDomainKYCDocument(m kycDocumentModel) domain.KYCDocument {
	return domain.KYCDocument{
		KYCDocumentID: m.KYCDocumentID, UserID: m.UserID, DocumentType: m.DocumentType, FileKey: m.FileKey,
		Status: m.Status, RejectionReason: m.RejectionReason, UploadedAt: m.UploadedAt,
		ReviewedAt: m.ReviewedAt, ReviewedBy: m.ReviewedBy,
	}
}

func toDomainUsernameHistory(m usernameHistoryModel) domain.UsernameHistory {
	return domain.UsernameHistory{
		HistoryID: m.HistoryID, UserID: m.UserID, OldUsername: m.OldUsername, NewUsername: m.NewUsername,
		ChangedAt: m.ChangedAt, RedirectExpiresAt: m.RedirectExpiresAt,
	}
}

func toDomainProfileStats(m profileStatsModel) domain.ProfileStats {
	return domain.ProfileStats{
		StatID: m.StatID, UserID: m.UserID, TotalEarningsYTD: m.TotalEarningsYTD,
		SubmissionCount: m.SubmissionCount, ApprovalRate: m.ApprovalRate, FollowerCount: m.FollowerCount,
		LastUpdatedAt: m.LastUpdatedAt,
	}
}
