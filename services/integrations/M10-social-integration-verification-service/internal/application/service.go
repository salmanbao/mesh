package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M10-social-integration-verification-service/internal/domain"
)

func (s *Service) ConnectStart(ctx context.Context, actor Actor, input ConnectInput) (ConnectResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return ConnectResult{}, domain.ErrUnauthorized
	}
	input.Provider = normalizeProvider(input.Provider)
	input.UserID = strings.TrimSpace(input.UserID)
	if input.UserID == "" || input.Provider == "" || !domain.IsValidProvider(input.Provider) {
		return ConnectResult{}, domain.ErrInvalidInput
	}
	if !canActForUser(actor, input.UserID) {
		return ConnectResult{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return ConnectResult{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashJSON(map[string]string{"op": "connect", "user_id": input.UserID, "provider": input.Provider})
	if cached, ok, err := s.getIdempotentBody(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return ConnectResult{}, err
	} else if ok {
		var out ConnectResult
		if err := json.Unmarshal(cached, &out); err == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return ConnectResult{}, err
	}
	state := uuid.NewString()
	res := ConnectResult{AuthURL: fmt.Sprintf("https://oauth.%s.example/authorize?client_id=mesh-demo&state=%s", input.Provider, state), State: state}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, res)
	return res, nil
}

func (s *Service) OAuthCallback(ctx context.Context, actor Actor, input CallbackInput) (domain.SocialAccount, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.SocialAccount{}, domain.ErrUnauthorized
	}
	input.Provider = normalizeProvider(input.Provider)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Code = strings.TrimSpace(input.Code)
	input.State = strings.TrimSpace(input.State)
	input.Handle = strings.TrimSpace(input.Handle)
	if input.UserID == "" || input.Provider == "" || input.Code == "" || input.State == "" || !domain.IsValidProvider(input.Provider) {
		return domain.SocialAccount{}, domain.ErrInvalidInput
	}
	if !canActForUser(actor, input.UserID) {
		return domain.SocialAccount{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.SocialAccount{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashJSON(input)
	if cached, ok, err := s.getIdempotentBody(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SocialAccount{}, err
	} else if ok {
		var out domain.SocialAccount
		if err := json.Unmarshal(cached, &out); err == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SocialAccount{}, err
	}
	now := s.nowFn()
	acc, err := s.accounts.GetByUserProvider(ctx, input.UserID, input.Provider)
	if err != nil {
		acc = domain.SocialAccount{SocialAccountID: uuid.NewString(), UserID: input.UserID, Provider: input.Provider, ConnectedAt: now}
	}
	exp := now.Add(24 * time.Hour)
	if input.Handle == "" {
		input.Handle = input.UserID + "_" + input.Provider
	}
	acc.Handle = input.Handle
	acc.Status = domain.AccountStatusActive
	acc.AccessToken = "enc_access_" + uuid.NewString()
	acc.RefreshToken = "enc_refresh_" + uuid.NewString()
	acc.TokenExpiresAt = &exp
	if acc.ConnectedAt.IsZero() {
		acc.ConnectedAt = now
	}
	acc.UpdatedAt = now
	if _, err2 := s.accounts.GetByID(ctx, acc.SocialAccountID); err2 == nil {
		if err := s.accounts.Update(ctx, acc); err != nil {
			return domain.SocialAccount{}, err
		}
	} else {
		if err := s.accounts.Create(ctx, acc); err != nil {
			return domain.SocialAccount{}, err
		}
	}
	if err := s.enqueueSocialAccountConnected(ctx, acc, actor.RequestID, now); err != nil {
		return domain.SocialAccount{}, err
	}
	if err := s.enqueueSocialStatusChanged(ctx, acc.UserID, acc.Provider, acc.Status, actor.RequestID, now); err != nil {
		return domain.SocialAccount{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, acc)
	return acc, nil
}

func (s *Service) ListAccounts(ctx context.Context, actor Actor, userID string) ([]domain.SocialAccount, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		userID = strings.TrimSpace(actor.SubjectID)
	}
	if !canActForUser(actor, userID) {
		return nil, domain.ErrForbidden
	}
	return s.accounts.ListByUserID(ctx, userID)
}

func (s *Service) DisconnectAccount(ctx context.Context, actor Actor, socialAccountID string) (domain.SocialAccount, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.SocialAccount{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.SocialAccount{}, domain.ErrIdempotencyRequired
	}
	socialAccountID = strings.TrimSpace(socialAccountID)
	if socialAccountID == "" {
		return domain.SocialAccount{}, domain.ErrInvalidInput
	}
	acc, err := s.accounts.GetByID(ctx, socialAccountID)
	if err != nil {
		return domain.SocialAccount{}, err
	}
	if !canActForUser(actor, acc.UserID) {
		return domain.SocialAccount{}, domain.ErrForbidden
	}
	requestHash := hashJSON(map[string]string{"op": "disconnect", "social_account_id": socialAccountID})
	if cached, ok, err := s.getIdempotentBody(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SocialAccount{}, err
	} else if ok {
		var out domain.SocialAccount
		if err := json.Unmarshal(cached, &out); err == nil { return out, nil }
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.SocialAccount{}, err
	}
	now := s.nowFn()
	acc.Status = domain.AccountStatusRevoked
	acc.AccessToken = ""
	acc.RefreshToken = ""
	acc.TokenExpiresAt = nil
	acc.UpdatedAt = now
	acc.DisconnectedAt = &now
	if err := s.accounts.Update(ctx, acc); err != nil {
		return domain.SocialAccount{}, err
	}
	if err := s.enqueueSocialStatusChanged(ctx, acc.UserID, acc.Provider, acc.Status, actor.RequestID, now); err != nil {
		return domain.SocialAccount{}, err
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, acc)
	return acc, nil
}

func (s *Service) RecordFollowersSync(ctx context.Context, actor Actor, input RecordFollowersSyncInput) (domain.SocialMetric, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.SocialMetric{}, domain.ErrUnauthorized
	}
	input.SocialAccountID = strings.TrimSpace(input.SocialAccountID)
	if input.SocialAccountID == "" || input.FollowerCount < 0 {
		return domain.SocialMetric{}, domain.ErrInvalidInput
	}
	acc, err := s.accounts.GetByID(ctx, input.SocialAccountID)
	if err != nil {
		return domain.SocialMetric{}, err
	}
	if !canActForUser(actor, acc.UserID) {
		return domain.SocialMetric{}, domain.ErrForbidden
	}
	now := s.nowFn()
	row := domain.SocialMetric{MetricID: uuid.NewString(), SocialAccountID: acc.SocialAccountID, UserID: acc.UserID, Provider: acc.Provider, FollowerCount: input.FollowerCount, SyncedAt: now}
	if err := s.metrics.Append(ctx, row); err != nil {
		return domain.SocialMetric{}, err
	}
	if err := s.enqueueFollowersSynced(ctx, row, actor.RequestID, now); err != nil {
		return domain.SocialMetric{}, err
	}
	return row, nil
}

func (s *Service) ValidatePost(ctx context.Context, actor Actor, input PostValidationInput) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ErrUnauthorized
	}
	input.UserID = strings.TrimSpace(input.UserID)
	input.Platform = normalizeProvider(input.Platform)
	input.PostID = strings.TrimSpace(input.PostID)
	if input.UserID == "" || input.PostID == "" || !domain.IsValidProvider(input.Platform) {
		return domain.ErrInvalidInput
	}
	if !canActForUser(actor, input.UserID) {
		return domain.ErrForbidden
	}
	return s.enqueuePostValidated(ctx, input, actor.RequestID, s.nowFn())
}

func (s *Service) ReportComplianceViolation(ctx context.Context, actor Actor, input ComplianceViolationInput) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ErrUnauthorized
	}
	input.UserID = strings.TrimSpace(input.UserID)
	input.Platform = normalizeProvider(input.Platform)
	input.PostID = strings.TrimSpace(input.PostID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.UserID == "" || input.PostID == "" || input.Reason == "" || !domain.IsValidProvider(input.Platform) {
		return domain.ErrInvalidInput
	}
	if !canActForUser(actor, input.UserID) {
		return domain.ErrForbidden
	}
	return s.enqueueComplianceViolation(ctx, input, actor.RequestID, s.nowFn())
}

func canActForUser(actor Actor, userID string) bool {
	if strings.TrimSpace(actor.SubjectID) == strings.TrimSpace(userID) {
		return true
	}
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "system"
}

func normalizeProvider(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "x" {
		return "twitter"
	}
	return v
}

func (s *Service) getIdempotentBody(ctx context.Context, key, requestHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != requestHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return append([]byte(nil), rec.ResponseBody...), true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	err := s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
	if err == domain.ErrConflict {
		return domain.ErrIdempotencyConflict
	}
	return err
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	b, _ := json.Marshal(payload)
	return s.idempotency.Complete(ctx, key, code, b, s.nowFn())
}

func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
