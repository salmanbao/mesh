package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/contracts"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M14-payout-settlement-service/internal/ports"
)

func (s *Service) RequestPayout(ctx context.Context, actor Actor, input RequestPayoutInput) (domain.Payout, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Payout{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" && actor.SubjectID != input.UserID {
		return domain.Payout{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Payout{}, domain.ErrIdempotencyRequired
	}
	return s.requestPayoutWithKey(ctx, actor, input, actor.IdempotencyKey)
}

func (s *Service) GetPayout(ctx context.Context, actor Actor, payoutID string) (domain.Payout, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Payout{}, domain.ErrUnauthorized
	}
	payout, err := s.payouts.GetByID(ctx, payoutID)
	if err != nil {
		return domain.Payout{}, err
	}
	if actor.Role != "admin" && payout.UserID != actor.SubjectID {
		return domain.Payout{}, domain.ErrForbidden
	}
	return payout, nil
}

func (s *Service) ListPayoutHistory(ctx context.Context, actor Actor, query ports.HistoryQuery) (HistoryOutput, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return HistoryOutput{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" {
		query.UserID = actor.SubjectID
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	items, total, err := s.payouts.List(ctx, query)
	if err != nil {
		return HistoryOutput{}, err
	}
	return HistoryOutput{
		Items: items,
		Pagination: contracts.Pagination{
			Limit:  query.Limit,
			Offset: query.Offset,
			Total:  total,
		},
	}, nil
}

func (s *Service) requestPayoutWithKey(ctx context.Context, actor Actor, input RequestPayoutInput, idempotencyKey string) (domain.Payout, error) {
	if input.Currency == "" {
		input.Currency = s.cfg.DefaultCurrency
	}
	if input.ScheduledAt.IsZero() {
		input.ScheduledAt = s.nowFn()
	}
	if err := domain.ValidatePayoutRequestInput(input.UserID, input.SubmissionID, input.Amount, input.Method, input.ScheduledAt); err != nil {
		return domain.Payout{}, err
	}
	if _, err := s.auth.GetUser(ctx, input.UserID); err != nil {
		return domain.Payout{}, fmt.Errorf("auth lookup: %w", err)
	}
	if err := s.profile.EnsurePayoutProfile(ctx, input.UserID); err != nil {
		return domain.Payout{}, fmt.Errorf("profile payout setup: %w", err)
	}
	if err := s.billing.EnsureBillingAccount(ctx, input.UserID); err != nil {
		return domain.Payout{}, fmt.Errorf("billing account check: %w", err)
	}
	if err := s.escrow.EnsureReleasable(ctx, input.SubmissionID); err != nil {
		return domain.Payout{}, fmt.Errorf("escrow hold check: %w", err)
	}
	if err := s.risk.EnsureEligible(ctx, input.UserID, input.Amount); err != nil {
		return domain.Payout{}, fmt.Errorf("risk gate: %w", err)
	}
	if err := s.finance.EnsureLiquidity(ctx, input.UserID, input.Amount, input.Currency); err != nil {
		return domain.Payout{}, fmt.Errorf("finance liquidity check: %w", err)
	}
	if err := s.reward.EnsureRewardEligible(ctx, input.SubmissionID, input.UserID, input.Amount); err != nil {
		return domain.Payout{}, fmt.Errorf("reward eligibility check: %w", err)
	}

	requestHash := hashPayload(input)
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return domain.Payout{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Payout{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Payout
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Payout{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Payout{}, err
	}

	payout := domain.Payout{
		PayoutID:     uuid.NewString(),
		UserID:       input.UserID,
		SubmissionID: input.SubmissionID,
		Amount:       input.Amount,
		Currency:     input.Currency,
		Method:       input.Method,
		Status:       domain.PayoutStatusScheduled,
		ScheduledAt:  input.ScheduledAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.payouts.Create(ctx, payout); err != nil {
		return domain.Payout{}, err
	}

	processingAt := s.nowFn()
	payout.Status = domain.PayoutStatusProcessing
	payout.ProcessingAt = &processingAt
	payout.UpdatedAt = processingAt
	if err := s.payouts.Update(ctx, payout); err != nil {
		return domain.Payout{}, err
	}
	if err := s.publishAnalyticsProcessing(ctx, payout); err != nil {
		return domain.Payout{}, err
	}

	switch {
	case payout.Method == domain.PayoutMethodInstant && payout.Amount > s.cfg.InstantPayoutLimit:
		failedAt := s.nowFn()
		payout.Status = domain.PayoutStatusFailed
		payout.FailureReason = "instant_limit_exceeded"
		payout.FailedAt = &failedAt
		payout.UpdatedAt = failedAt
		if err := s.payouts.Update(ctx, payout); err != nil {
			return domain.Payout{}, err
		}
		if err := s.enqueueDomainPayoutFailed(ctx, payout); err != nil {
			return domain.Payout{}, err
		}
	default:
		paidAt := s.nowFn()
		payout.Status = domain.PayoutStatusPaid
		payout.PaidAt = &paidAt
		payout.UpdatedAt = paidAt
		if err := s.payouts.Update(ctx, payout); err != nil {
			return domain.Payout{}, err
		}
		if err := s.enqueueDomainPayoutPaid(ctx, payout); err != nil {
			return domain.Payout{}, err
		}
	}

	if err := s.FlushOutbox(ctx); err != nil {
		return domain.Payout{}, err
	}
	payload, err := json.Marshal(payout)
	if err != nil {
		return domain.Payout{}, err
	}
	if err := s.idempotency.Complete(ctx, idempotencyKey, 201, payload, s.nowFn()); err != nil {
		return domain.Payout{}, err
	}
	_ = actor
	return payout, nil
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
