package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/contracts"
	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/domain"
)

func (s *Service) ValidatePost(ctx context.Context, actor Actor, in ValidatePostInput) (contracts.ValidatePostResponse, error) {
	uid, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return contracts.ValidatePostResponse{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return contracts.ValidatePostResponse{}, domain.ErrIdempotencyRequired
	}
	platform := strings.ToLower(strings.TrimSpace(in.Platform))
	normURL, valid := normalizePostURL(in.PostURL)
	if uid == "" || !domain.IsValidPlatform(platform) || !valid {
		return contracts.ValidatePostResponse{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "validate_post", "user_id": uid, "platform": platform, "post_url": normURL})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.ValidatePostResponse{}, err
	} else if ok {
		var out contracts.ValidatePostResponse
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return contracts.ValidatePostResponse{}, err
	}
	out := contracts.ValidatePostResponse{Valid: true, Platform: platform, NormalizedURL: normURL}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, out)
	return out, nil
}

func (s *Service) RegisterPost(ctx context.Context, actor Actor, in RegisterPostInput) (domain.TrackedPost, bool, error) {
	uid, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.TrackedPost{}, false, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.TrackedPost{}, false, domain.ErrIdempotencyRequired
	}
	platform := strings.ToLower(strings.TrimSpace(in.Platform))
	normURL, valid := normalizePostURL(in.PostURL)
	if uid == "" || !domain.IsValidPlatform(platform) || !valid {
		return domain.TrackedPost{}, false, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "register_post", "user_id": uid, "platform": platform, "post_url": normURL, "distribution_item_id": strings.TrimSpace(in.DistributionItemID), "campaign_id": strings.TrimSpace(in.CampaignID)})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.TrackedPost{}, false, err
	} else if ok {
		var out domain.TrackedPost
		if json.Unmarshal(raw, &out) == nil {
			return out, false, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.TrackedPost{}, false, err
	}
	if existing, err := s.posts.FindByUserPlatformURL(ctx, uid, platform, normURL); err == nil {
		_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, existing)
		return existing, false, nil
	}
	now := s.nowFn()
	attributionPending := strings.TrimSpace(in.DistributionItemID) == ""
	status := domain.TrackedPostStatusActive
	if attributionPending {
		status = domain.TrackedPostStatusPendingAttribution
	}
	post := domain.TrackedPost{TrackedPostID: "tp-" + uuid.NewString(), UserID: uid, Platform: platform, PostURL: normURL, DistributionItemID: strings.TrimSpace(in.DistributionItemID), CampaignID: strings.TrimSpace(in.CampaignID), Status: status, ValidationStatus: "validated", CreatedAt: now, UpdatedAt: now}
	if err := s.posts.Create(ctx, post); err != nil {
		return domain.TrackedPost{}, false, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, post)
	return post, attributionPending, nil
}

func (s *Service) GetTrackedPost(ctx context.Context, actor Actor, trackedPostID string) (domain.TrackedPost, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.TrackedPost{}, domain.ErrUnauthorized
	}
	trackedPostID = strings.TrimSpace(trackedPostID)
	if trackedPostID == "" {
		return domain.TrackedPost{}, domain.ErrInvalidInput
	}
	row, err := s.posts.GetByID(ctx, trackedPostID)
	if err != nil {
		return domain.TrackedPost{}, err
	}
	if !canActForUser(actor, row.UserID) {
		return domain.TrackedPost{}, domain.ErrForbidden
	}
	return row, nil
}

func (s *Service) GetTrackedPostMetrics(ctx context.Context, actor Actor, trackedPostID string) (domain.TrackedPost, []domain.MetricSnapshot, error) {
	post, err := s.GetTrackedPost(ctx, actor, trackedPostID)
	if err != nil {
		return domain.TrackedPost{}, nil, err
	}
	snaps, err := s.snapshots.ListByTrackedPostID(ctx, trackedPostID)
	return post, snaps, err
}

func (s *Service) RunPollCycle(ctx context.Context) error {
	if s.posts == nil || s.snapshots == nil {
		return nil
	}
	before := s.nowFn().Add(-s.cfg.PollCadence)
	posts, err := s.posts.ListPollCandidates(ctx, before, 100)
	if err != nil {
		return err
	}
	for _, post := range posts {
		if post.Status == domain.TrackedPostStatusArchived {
			continue
		}
		if err := s.pollOnePost(ctx, post); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) pollOnePost(ctx context.Context, post domain.TrackedPost) error {
	now := s.nowFn()
	latest, err := s.snapshots.LatestByTrackedPostID(ctx, post.TrackedPostID)
	views, likes, shares, comments := 100, 10, 2, 1
	if err == nil {
		views = latest.Views + 50
		likes = latest.Likes + 5
		shares = latest.Shares + 1
		comments = latest.Comments + 1
	}
	snap := domain.MetricSnapshot{SnapshotID: "ms-" + uuid.NewString(), TrackedPostID: post.TrackedPostID, Platform: post.Platform, Views: views, Likes: likes, Shares: shares, Comments: comments, PolledAt: now}
	if err := s.snapshots.Append(ctx, snap); err != nil {
		return err
	}
	post.LastPolledAt = &now
	post.UpdatedAt = now
	if post.Status == domain.TrackedPostStatusPendingAttribution && strings.TrimSpace(post.DistributionItemID) != "" {
		post.Status = domain.TrackedPostStatusActive
	}
	if err := s.posts.Update(ctx, post); err != nil {
		return err
	}
	return s.enqueueTrackingMetricsUpdated(ctx, snap, post, now)
}

func normalizePostURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	u.Fragment = ""
	return strings.ToLower(u.String()), true
}

func canActForUser(actor Actor, userID string) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	actorID := strings.TrimSpace(actor.SubjectID)
	userID = strings.TrimSpace(userID)
	return actorID != "" && userID != "" && (actorID == userID || role == "admin" || role == "support")
}

func (s *Service) resolveUser(actor Actor, requested string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if !canActForUser(actor, requested) {
		return "", domain.ErrForbidden
	}
	return requested, nil
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) getIdempotent(ctx context.Context, key, expectedHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != expectedHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return rec.ResponseBody, true, nil
}
func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}
func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, v any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(v)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}
