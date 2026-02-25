package application

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

var allowedAvatarMIMEs = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
}

var allowedKYCMIMEs = map[string]struct{}{
	"application/pdf": {},
	"image/jpeg":      {},
	"image/png":       {},
}

type PublicProfileLookup struct {
	Profile    PublicProfileResponse
	RedirectTo string
}

func (s *Service) GetMyProfile(ctx context.Context, userID uuid.UUID) (ProfileResponse, error) {
	rm, err := s.reads.GetProfileReadModelByUserID(ctx, userID)
	if err != nil {
		return ProfileResponse{}, err
	}
	return toProfileResponse(rm, true), nil
}

func (s *Service) GetPublicProfile(ctx context.Context, username string, requester *uuid.UUID) (PublicProfileLookup, error) {
	username = domain.NormalizeUsername(username)
	if err := domain.ValidateUsername(username); err != nil {
		return PublicProfileLookup{}, err
	}

	rm, found, err := s.reads.GetPublicProfileByUsername(ctx, username, s.nowFn())
	if err != nil {
		return PublicProfileLookup{}, err
	}
	if !found {
		redirectTo, redirectFound, rErr := s.usernameHistory.ResolveRedirect(ctx, username, s.nowFn())
		if rErr != nil {
			return PublicProfileLookup{}, rErr
		}
		if redirectFound {
			return PublicProfileLookup{RedirectTo: redirectTo}, nil
		}
		return PublicProfileLookup{}, domain.ErrNotFound
	}

	p := rm.Profile
	resp := PublicProfileResponse{
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarURL:   p.AvatarURL,
		MemberSince: p.CreatedAt,
	}

	isOwner := requester != nil && *requester == p.UserID
	if p.IsPrivate && !isOwner {
		resp.IsPrivate = true
		resp.Message = "This profile is private"
		resp.Bio = ""
		resp.Statistics = nil
		resp.SocialLinks = nil
		return PublicProfileLookup{Profile: resp}, nil
	}

	for _, sl := range rm.SocialLinks {
		resp.SocialLinks = append(resp.SocialLinks, SocialLinkView{
			Platform: sl.Platform, ProfileURL: sl.ProfileURL, Verified: sl.Verified, Handle: sl.Handle,
		})
	}
	if p.HideStatistics && !isOwner {
		resp.Statistics = nil
	} else {
		resp.Statistics = toProfileStatsView(rm.Stats)
	}
	return PublicProfileLookup{Profile: resp}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest, idempotencyKey string) (ProfileResponse, error) {
	if req.DisplayName != nil {
		if err := domain.ValidateDisplayName(*req.DisplayName); err != nil {
			return ProfileResponse{}, err
		}
	}
	if req.Bio != nil {
		if err := domain.ValidateBio(*req.Bio); err != nil {
			return ProfileResponse{}, err
		}
	}
	if err := s.reserveIdempotency(ctx, idempotencyKey, req); err != nil {
		return ProfileResponse{}, err
	}

	updated, err := s.profiles.UpdateProfile(ctx, ports.UpdateProfileParams{
		UserID:          userID,
		DisplayName:     req.DisplayName,
		Bio:             req.Bio,
		IsPrivate:       req.IsPrivate,
		IsUnlisted:      req.IsUnlisted,
		HideStatistics:  req.HideStatistics,
		AnalyticsOptOut: req.AnalyticsOptOut,
		UpdatedAt:       s.nowFn(),
	})
	if err != nil {
		return ProfileResponse{}, err
	}
	_ = s.enqueueProfileUpdated(ctx, updated)
	_ = s.cache.Delete(ctx, cacheKeyUser(userID), cacheKeyUsername(updated.Username))

	rm, err := s.reads.GetProfileReadModelByUserID(ctx, userID)
	if err != nil {
		return ProfileResponse{}, err
	}
	return toProfileResponse(rm, true), nil
}

func (s *Service) ChangeUsername(ctx context.Context, userID uuid.UUID, req ChangeUsernameRequest, idempotencyKey string) (ProfileResponse, error) {
	if err := domain.ValidateUsername(req.Username); err != nil {
		return ProfileResponse{}, err
	}
	newUsername := domain.NormalizeUsername(req.Username)
	if err := s.reserveIdempotency(ctx, idempotencyKey, map[string]string{"username": newUsername}); err != nil {
		return ProfileResponse{}, err
	}

	reserved, err := s.reservedUsernames.IsReserved(ctx, newUsername)
	if err != nil {
		return ProfileResponse{}, err
	}
	if reserved {
		return ProfileResponse{}, fmt.Errorf("%w: username reserved", domain.ErrConflict)
	}

	profile, err := s.profiles.GetByUserID(ctx, userID)
	if err != nil {
		return ProfileResponse{}, err
	}
	if profile.LastUsernameChangeAt != nil {
		nextAllowed := profile.LastUsernameChangeAt.Add(time.Duration(s.cfg.UsernameCooldownDays) * 24 * time.Hour)
		if nextAllowed.After(s.nowFn()) {
			return ProfileResponse{}, fmt.Errorf("%w: username change cooldown active", domain.ErrForbidden)
		}
	}

	oldUsername, updated, err := s.profiles.UpdateUsername(ctx, userID, newUsername, s.nowFn(), s.cfg.UsernameRedirectDays)
	if err != nil {
		return ProfileResponse{}, err
	}
	_ = s.enqueueProfileUpdated(ctx, updated)
	_ = s.cache.Delete(ctx, cacheKeyUser(userID), cacheKeyUsername(oldUsername), cacheKeyUsername(newUsername))

	rm, err := s.reads.GetProfileReadModelByUserID(ctx, userID)
	if err != nil {
		return ProfileResponse{}, err
	}
	resp := toProfileResponse(rm, true)
	resp.Message = "username updated"
	return resp, nil
}

func (s *Service) CheckUsernameAvailability(ctx context.Context, username string) (UsernameAvailabilityResponse, error) {
	if err := domain.ValidateUsername(username); err != nil {
		return UsernameAvailabilityResponse{
			Username:  domain.NormalizeUsername(username),
			Available: false,
			Reason:    "invalid",
		}, nil
	}
	username = domain.NormalizeUsername(username)
	reserved, err := s.reservedUsernames.IsReserved(ctx, username)
	if err != nil {
		return UsernameAvailabilityResponse{}, err
	}
	if reserved {
		return UsernameAvailabilityResponse{Username: username, Available: false, Reason: "reserved"}, nil
	}
	state, err := s.profiles.CheckUsernameAvailability(ctx, username)
	if err != nil {
		return UsernameAvailabilityResponse{}, err
	}
	return UsernameAvailabilityResponse{Username: username, Available: state.Available, Reason: state.Reason}, nil
}

func (s *Service) AddSocialLink(ctx context.Context, userID uuid.UUID, req AddSocialLinkRequest, idempotencyKey string) (SocialLinkView, error) {
	if err := s.reserveIdempotency(ctx, idempotencyKey, req); err != nil {
		return SocialLinkView{}, err
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	handle, err := domain.ValidateSocialURL(platform, strings.TrimSpace(req.ProfileURL))
	if err != nil {
		return SocialLinkView{}, err
	}
	count, err := s.socialLinks.CountByUserID(ctx, userID)
	if err != nil {
		return SocialLinkView{}, err
	}
	if count >= int64(s.cfg.MaxSocialLinks) {
		return SocialLinkView{}, fmt.Errorf("%w: max social links reached", domain.ErrInvalidInput)
	}

	var oauthID *uuid.UUID
	if req.OAuthConnectionID != nil && *req.OAuthConnectionID != "" {
		parsed, pErr := uuid.Parse(*req.OAuthConnectionID)
		if pErr != nil {
			return SocialLinkView{}, fmt.Errorf("%w: invalid oauth_connection_id", domain.ErrInvalidInput)
		}
		oauthID = &parsed
	}
	verified := platform == "twitter" || platform == "linkedin" || platform == "website"
	if !verified && oauthID == nil {
		return SocialLinkView{}, fmt.Errorf("%w: oauth verification required", domain.ErrInvalidInput)
	}
	if oauthID != nil {
		verified = true
	}

	created, err := s.socialLinks.Create(ctx, ports.CreateSocialLinkParams{
		UserID:            userID,
		Platform:          platform,
		Handle:            handle,
		ProfileURL:        req.ProfileURL,
		Verified:          verified,
		OAuthConnectionID: oauthID,
		AddedAt:           s.nowFn(),
	})
	if err != nil {
		return SocialLinkView{}, err
	}
	profile, _ := s.profiles.GetByUserID(ctx, userID)
	_ = s.enqueueProfileUpdated(ctx, profile)
	_ = s.cache.Delete(ctx, cacheKeyUser(userID), cacheKeyUsername(profile.Username))

	return SocialLinkView{
		Platform: created.Platform, Handle: created.Handle, ProfileURL: created.ProfileURL, Verified: created.Verified,
	}, nil
}

func (s *Service) DeleteSocialLink(ctx context.Context, userID uuid.UUID, platform string) error {
	if err := s.socialLinks.DeleteByUserAndPlatform(ctx, userID, strings.ToLower(strings.TrimSpace(platform))); err != nil {
		return err
	}
	profile, _ := s.profiles.GetByUserID(ctx, userID)
	_ = s.enqueueProfileUpdated(ctx, profile)
	_ = s.cache.Delete(ctx, cacheKeyUser(userID), cacheKeyUsername(profile.Username))
	return nil
}

func (s *Service) PutPayoutMethod(ctx context.Context, userID uuid.UUID, req PutPayoutMethodRequest, idempotencyKey string) (PayoutMethodView, error) {
	if err := s.reserveIdempotency(ctx, idempotencyKey, req); err != nil {
		return PayoutMethodView{}, err
	}
	methodType := strings.ToLower(strings.TrimSpace(req.MethodType))
	raw := strings.TrimSpace(req.StripeAccountID)
	switch methodType {
	case "paypal":
		raw = strings.TrimSpace(req.Email)
	case "usdc_polygon", "btc", "eth":
		raw = strings.TrimSpace(req.WalletAddress)
		if strings.TrimSpace(req.WalletAddressConfirmation) != raw {
			return PayoutMethodView{}, fmt.Errorf("%w: wallet confirmation mismatch", domain.ErrInvalidInput)
		}
	}
	if err := domain.ValidatePayoutMethodInput(methodType, raw); err != nil {
		return PayoutMethodView{}, err
	}
	encrypted, err := s.encryption.Encrypt(userID.String(), raw)
	if err != nil {
		return PayoutMethodView{}, err
	}
	pm, err := s.payoutMethods.Upsert(ctx, ports.PutPayoutMethodParams{
		UserID:              userID,
		MethodType:          methodType,
		IdentifierEncrypted: encrypted,
		VerificationStatus:  "unverified",
		Now:                 s.nowFn(),
	})
	if err != nil {
		return PayoutMethodView{}, err
	}
	profile, _ := s.profiles.GetByUserID(ctx, userID)
	_ = s.enqueueProfileUpdated(ctx, profile)
	return PayoutMethodView{
		MethodType: pm.MethodType, VerificationStatus: pm.VerificationStatus, LastUsedAt: pm.LastUsedAt,
	}, nil
}

func (s *Service) UploadAvatar(ctx context.Context, userID uuid.UUID, fileName, contentType string, fileBytes []byte, idempotencyKey string) (AvatarUploadResponse, error) {
	if err := s.reserveIdempotency(ctx, idempotencyKey, map[string]any{"name": fileName, "len": len(fileBytes), "content_type": contentType}); err != nil {
		return AvatarUploadResponse{}, err
	}
	if len(fileBytes) == 0 || len(fileBytes) > 5*1024*1024 {
		return AvatarUploadResponse{}, fmt.Errorf("%w: file size exceeds 5MB limit", domain.ErrInvalidInput)
	}
	if _, ok := allowedAvatarMIMEs[contentType]; !ok {
		return AvatarUploadResponse{}, fmt.Errorf("%w: file must be JPG or PNG format", domain.ErrInvalidInput)
	}

	uploadID := uuid.NewString()
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".jpg"
	}
	avatarURL := fmt.Sprintf("https://cdn.viralforge.com/avatars/%s/avatar_200_%s%s", userID.String(), uploadID, ext)
	bannerURL := fmt.Sprintf("https://cdn.viralforge.com/avatars/%s/banner_800_%s%s", userID.String(), uploadID, ext)
	_, err := s.profiles.UpdateProfile(ctx, ports.UpdateProfileParams{
		UserID:    userID,
		AvatarURL: &avatarURL,
		BannerURL: &bannerURL,
		UpdatedAt: s.nowFn(),
	})
	if err != nil {
		return AvatarUploadResponse{}, err
	}
	profile, _ := s.profiles.GetByUserID(ctx, userID)
	_ = s.enqueueProfileUpdated(ctx, profile)
	_ = s.cache.Delete(ctx, cacheKeyUser(userID), cacheKeyUsername(profile.Username))

	return AvatarUploadResponse{
		UploadID: uploadID,
		Status:   "processing",
		Message:  "Avatar is being processed. You'll be notified when it's ready.",
	}, nil
}

func (s *Service) UploadKYCDocument(ctx context.Context, userID uuid.UUID, req UploadKYCDocumentRequest, idempotencyKey string) (KYCDocumentView, error) {
	if err := s.reserveIdempotency(ctx, idempotencyKey, map[string]any{"document_type": req.DocumentType, "size": len(req.FileBytes)}); err != nil {
		return KYCDocumentView{}, err
	}
	if len(req.FileBytes) == 0 || len(req.FileBytes) > 10*1024*1024 {
		return KYCDocumentView{}, fmt.Errorf("%w: file size exceeds 10MB limit", domain.ErrInvalidInput)
	}
	if _, ok := allowedKYCMIMEs[req.FileContentType]; !ok {
		return KYCDocumentView{}, fmt.Errorf("%w: invalid file type", domain.ErrInvalidInput)
	}
	documentType := strings.ToLower(strings.TrimSpace(req.DocumentType))
	switch documentType {
	case "passport", "government_id", "drivers_license", "proof_of_address":
	default:
		return KYCDocumentView{}, fmt.Errorf("%w: unsupported document_type", domain.ErrInvalidInput)
	}

	docID := uuid.New()
	fileKey := fmt.Sprintf("viralforge-kyc-documents/%s/%s", userID.String(), docID.String())
	doc, err := s.kyc.CreateDocument(ctx, ports.CreateKYCDocumentParams{
		UserID:       userID,
		DocumentType: documentType,
		FileKey:      fileKey,
		Status:       "uploaded",
		UploadedAt:   s.nowFn(),
	})
	if err != nil {
		return KYCDocumentView{}, err
	}
	_, _ = s.profiles.UpdateProfile(ctx, ports.UpdateProfileParams{UserID: userID, UpdatedAt: s.nowFn()})
	_ = s.kyc.UpdateStatus(ctx, userID, domain.KYCStatusPending, "", s.nowFn(), nil)

	return KYCDocumentView{
		DocumentType: doc.DocumentType,
		Status:       doc.Status,
		UploadedAt:   doc.UploadedAt,
	}, nil
}

func (s *Service) GetKYCStatus(ctx context.Context, userID uuid.UUID) (ProfileResponse, error) {
	rm, err := s.reads.GetProfileReadModelByUserID(ctx, userID)
	if err != nil {
		return ProfileResponse{}, err
	}
	resp := ProfileResponse{
		KYCStatus: string(rm.Profile.KYCStatus),
		Documents: make([]KYCDocumentView, 0, len(rm.Documents)),
	}
	for _, doc := range rm.Documents {
		resp.Documents = append(resp.Documents, KYCDocumentView{
			DocumentType:    doc.DocumentType,
			Status:          doc.Status,
			UploadedAt:      doc.UploadedAt,
			ReviewedAt:      doc.ReviewedAt,
			RejectionReason: doc.RejectionReason,
		})
	}
	return resp, nil
}

func (s *Service) AdminListProfiles(ctx context.Context, limit, offset int) ([]ProfileResponse, error) {
	pending, err := s.kyc.ListPendingQueue(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]ProfileResponse, 0, len(pending))
	for _, p := range pending {
		out = append(out, ProfileResponse{
			ProfileID:   p.ProfileID.String(),
			UserID:      p.UserID.String(),
			Username:    p.Username,
			DisplayName: p.DisplayName,
			KYCStatus:   string(p.KYCStatus),
			CreatedAt:   p.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) AdminApproveKYC(ctx context.Context, req AdminKYCDecisionRequest) error {
	if err := s.kyc.UpdateStatus(ctx, req.UserID, domain.KYCStatusVerified, "", req.Now, &req.ReviewedBy); err != nil {
		return err
	}
	_, _ = s.profiles.UpdateProfile(ctx, ports.UpdateProfileParams{UserID: req.UserID, UpdatedAt: req.Now})
	return nil
}

func (s *Service) AdminRejectKYC(ctx context.Context, req AdminKYCDecisionRequest) error {
	if strings.TrimSpace(req.RejectionReason) == "" {
		return fmt.Errorf("%w: rejection_reason required", domain.ErrInvalidInput)
	}
	if err := s.kyc.UpdateStatus(ctx, req.UserID, domain.KYCStatusRejected, req.RejectionReason, req.Now, &req.ReviewedBy); err != nil {
		return err
	}
	_, _ = s.profiles.UpdateProfile(ctx, ports.UpdateProfileParams{UserID: req.UserID, UpdatedAt: req.Now})
	return nil
}

type userRegisteredEvent struct {
	EventID string `json:"event_id"`
	Data    struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

type userDeletedEvent struct {
	EventID string `json:"event_id"`
	Data    struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (s *Service) HandleUserRegistered(ctx context.Context, payload []byte) error {
	var evt userRegisteredEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("%w: invalid user.registered payload", domain.ErrInvalidInput)
	}
	dup, err := s.eventDedup.IsDuplicate(ctx, evt.EventID, s.nowFn())
	if err != nil {
		return err
	}
	if dup {
		return nil
	}
	userID, err := uuid.Parse(evt.Data.UserID)
	if err != nil {
		return fmt.Errorf("%w: invalid user_id", domain.ErrInvalidInput)
	}
	identity, err := s.authClient.GetUserIdentity(ctx, userID)
	if err != nil {
		return err
	}
	displayName := sanitizeDisplayNameFromEmail(identity.Email)
	created, err := s.profiles.CreateProfileWithDefaults(ctx, ports.CreateProfileParams{
		UserID:      userID,
		DisplayName: displayName,
		CreatedAt:   s.nowFn(),
	})
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return err
	}
	_ = s.stats.Upsert(ctx, ports.UpsertProfileStatsParams{
		UserID:    userID,
		UpdatedAt: s.nowFn(),
	})
	_ = s.eventDedup.MarkProcessed(ctx, evt.EventID, "user.registered", s.nowFn().Add(s.cfg.EventDedupTTL))
	if created.UserID != uuid.Nil {
		_ = s.enqueueProfileUpdated(ctx, created)
	}
	return nil
}

func (s *Service) HandleUserDeleted(ctx context.Context, payload []byte) error {
	var evt userDeletedEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return fmt.Errorf("%w: invalid user.deleted payload", domain.ErrInvalidInput)
	}
	dup, err := s.eventDedup.IsDuplicate(ctx, evt.EventID, s.nowFn())
	if err != nil {
		return err
	}
	if dup {
		return nil
	}
	userID, err := uuid.Parse(evt.Data.UserID)
	if err != nil {
		return fmt.Errorf("%w: invalid user_id", domain.ErrInvalidInput)
	}
	if err := s.profiles.SoftDeleteByUserID(ctx, userID, s.nowFn()); err != nil {
		return err
	}
	_ = s.eventDedup.MarkProcessed(ctx, evt.EventID, "user.deleted", s.nowFn().Add(s.cfg.EventDedupTTL))
	return nil
}

func toProfileResponse(rm ports.ProfileReadModel, includePrivate bool) ProfileResponse {
	resp := ProfileResponse{
		ProfileID:       rm.Profile.ProfileID.String(),
		UserID:          rm.Profile.UserID.String(),
		Username:        rm.Profile.Username,
		DisplayName:     rm.Profile.DisplayName,
		Bio:             rm.Profile.Bio,
		AvatarURL:       rm.Profile.AvatarURL,
		BannerURL:       rm.Profile.BannerURL,
		KYCStatus:       string(rm.Profile.KYCStatus),
		IsPrivate:       rm.Profile.IsPrivate,
		IsUnlisted:      rm.Profile.IsUnlisted,
		HideStatistics:  rm.Profile.HideStatistics,
		AnalyticsOptOut: rm.Profile.AnalyticsOptOut,
		CreatedAt:       rm.Profile.CreatedAt,
		UpdatedAt:       rm.Profile.UpdatedAt,
		SocialLinks:     make([]SocialLinkView, 0, len(rm.SocialLinks)),
		PayoutMethods:   make([]PayoutMethodView, 0, len(rm.PayoutMethods)),
		Documents:       make([]KYCDocumentView, 0, len(rm.Documents)),
		Statistics:      toProfileStatsView(rm.Stats),
	}
	for _, sl := range rm.SocialLinks {
		resp.SocialLinks = append(resp.SocialLinks, SocialLinkView{
			Platform:   sl.Platform,
			Handle:     sl.Handle,
			ProfileURL: sl.ProfileURL,
			Verified:   sl.Verified,
		})
	}
	for _, pm := range rm.PayoutMethods {
		resp.PayoutMethods = append(resp.PayoutMethods, PayoutMethodView{
			MethodType: pm.MethodType, VerificationStatus: pm.VerificationStatus, LastUsedAt: pm.LastUsedAt,
		})
	}
	for _, doc := range rm.Documents {
		resp.Documents = append(resp.Documents, KYCDocumentView{
			DocumentType:    doc.DocumentType,
			Status:          doc.Status,
			UploadedAt:      doc.UploadedAt,
			ReviewedAt:      doc.ReviewedAt,
			RejectionReason: doc.RejectionReason,
		})
	}
	if !includePrivate {
		resp.PayoutMethods = nil
		resp.Documents = nil
	}
	return resp
}

func cacheKeyUser(userID uuid.UUID) string {
	return "profile:user:" + userID.String()
}

func cacheKeyUsername(username string) string {
	return "profile:username:" + strings.ToLower(strings.TrimSpace(username))
}

var unsafeDisplayRunes = regexp.MustCompile(`[^a-zA-Z0-9 _-]+`)

func sanitizeDisplayNameFromEmail(email string) string {
	local := strings.TrimSpace(strings.Split(email, "@")[0])
	if local == "" {
		local = "user"
	}
	local = unsafeDisplayRunes.ReplaceAllString(local, "")
	local = strings.TrimSpace(local)
	if len(local) < 3 {
		local = local + " user"
	}
	if len(local) > 50 {
		local = local[:50]
	}
	return local
}
