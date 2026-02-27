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
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/domain"
	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/ports"
)

func (s *Service) CalculateReward(ctx context.Context, actor Actor, input CalculateRewardInput) (domain.Reward, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Reward{}, domain.ErrUnauthorized
	}
	if actor.Role != "admin" && actor.Role != "finance" && actor.SubjectID != input.UserID {
		return domain.Reward{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Reward{}, domain.ErrIdempotencyRequired
	}
	return s.calculateWithKey(ctx, input, actor.IdempotencyKey)
}

func (s *Service) GetReward(ctx context.Context, actor Actor, submissionID string) (domain.Reward, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Reward{}, domain.ErrUnauthorized
	}
	reward, err := s.rewards.GetBySubmissionID(ctx, strings.TrimSpace(submissionID))
	if err != nil {
		return domain.Reward{}, err
	}
	if actor.Role != "admin" && actor.Role != "finance" && reward.UserID != actor.SubjectID {
		return domain.Reward{}, domain.ErrForbidden
	}
	return reward, nil
}

func (s *Service) GetRollover(ctx context.Context, actor Actor, userID string) (domain.RolloverBalance, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.RolloverBalance{}, domain.ErrUnauthorized
	}
	userID = strings.TrimSpace(userID)
	if actor.Role != "admin" && actor.Role != "finance" && actor.SubjectID != userID {
		return domain.RolloverBalance{}, domain.ErrForbidden
	}
	return s.rollovers.GetByUser(ctx, userID)
}

func (s *Service) ListRewardsByUser(ctx context.Context, actor Actor, userID string, limit, offset int) (RewardHistoryOutput, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return RewardHistoryOutput{}, domain.ErrUnauthorized
	}
	userID = strings.TrimSpace(userID)
	if actor.Role != "admin" && actor.Role != "finance" {
		userID = actor.SubjectID
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, total, err := s.rewards.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return RewardHistoryOutput{}, err
	}
	return RewardHistoryOutput{Items: items, Total: total}, nil
}

func (s *Service) calculateWithKey(ctx context.Context, input CalculateRewardInput, idempotencyKey string) (domain.Reward, error) {
	now := s.nowFn()
	if input.VerificationCompletedAt.IsZero() {
		input.VerificationCompletedAt = now
	}
	if input.RatePer1K <= 0 {
		rate, err := s.campaign.GetRatePer1K(ctx, input.CampaignID, input.UserID)
		if err != nil {
			return domain.Reward{}, fmt.Errorf("campaign rate lookup: %w", err)
		}
		input.RatePer1K = rate
	}
	if input.RatePer1K <= 0 {
		input.RatePer1K = s.cfg.DefaultRatePer1K
	}
	if input.LockedViews == 0 {
		snapshot, err := s.snapshots.Get(ctx, input.SubmissionID)
		if err == nil {
			input.LockedViews = snapshot.Views
		}
	}
	if input.LockedViews == 0 {
		views, err := s.tracking.GetLockedViews(ctx, input.SubmissionID)
		if err != nil {
			return domain.Reward{}, fmt.Errorf("tracking lookup: %w", err)
		}
		input.LockedViews = views
	}
	if err := domain.ValidateCalculationInput(input.UserID, input.SubmissionID, input.CampaignID, input.LockedViews, input.RatePer1K); err != nil {
		return domain.Reward{}, err
	}
	if _, err := s.auth.GetUser(ctx, input.UserID); err != nil {
		return domain.Reward{}, fmt.Errorf("auth lookup: %w", err)
	}
	if err := s.submission.ValidateSubmission(ctx, input.SubmissionID, input.UserID, input.CampaignID); err != nil {
		return domain.Reward{}, fmt.Errorf("submission validation: %w", err)
	}
	if input.FraudScore == 0 {
		score, err := s.voting.GetFraudScore(ctx, input.SubmissionID, input.UserID)
		if err != nil {
			return domain.Reward{}, fmt.Errorf("voting fraud score: %w", err)
		}
		input.FraudScore = score
	}

	requestHash := hashPayload(input)
	existing, err := s.idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return domain.Reward{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.Reward{}, domain.ErrIdempotencyConflict
		}
		var cached domain.Reward
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Reward{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Reward{}, err
	}

	gross := domain.CalculateGrossAmount(input.LockedViews, input.RatePer1K)
	rollover, err := s.rollovers.GetByUser(ctx, input.UserID)
	if err != nil {
		return domain.Reward{}, err
	}

	status := domain.RewardStatusCalculated
	rolloverApplied := 0.0
	nextBalance := rollover.Balance
	netAmount := gross
	var eligibleAt *time.Time

	if input.FraudScore >= s.cfg.FraudRejectThreshold {
		status = domain.RewardStatusFraudRejected
		netAmount = 0
		gross = 0
	} else {
		total := domain.RoundCurrency(gross+rollover.Balance, 4)
		if total < s.cfg.MinimumPayoutThreshold && total < s.cfg.MaxRolloverBalance {
			status = domain.RewardStatusBelowThreshold
			nextBalance = total
		} else {
			status = domain.RewardStatusEligible
			rolloverApplied = rollover.Balance
			nextBalance = 0
			netAmount = total
			nowEligible := now
			eligibleAt = &nowEligible
		}
	}

	reward := domain.Reward{
		SubmissionID:            input.SubmissionID,
		UserID:                  input.UserID,
		CampaignID:              input.CampaignID,
		LockedViews:             input.LockedViews,
		RatePer1K:               input.RatePer1K,
		GrossAmount:             gross,
		NetAmount:               netAmount,
		RolloverApplied:         rolloverApplied,
		RolloverBalance:         nextBalance,
		FraudScore:              input.FraudScore,
		Status:                  status,
		VerificationCompletedAt: input.VerificationCompletedAt,
		CalculatedAt:            now,
		EligibleAt:              eligibleAt,
		LastEventID:             input.EventID,
	}
	if err := s.rewards.Save(ctx, reward); err != nil {
		return domain.Reward{}, err
	}
	if status != domain.RewardStatusFraudRejected {
		if err := s.rollovers.Upsert(ctx, domain.RolloverBalance{UserID: input.UserID, Balance: nextBalance, UpdatedAt: now}); err != nil {
			return domain.Reward{}, err
		}
	}
	if err := s.audit.Append(ctx, ports.AuditRecord{
		LogID:        uuid.NewString(),
		SubmissionID: input.SubmissionID,
		UserID:       input.UserID,
		Action:       "reward_calculated",
		Amount:       netAmount,
		CreatedAt:    now,
		Metadata: map[string]string{
			"status": string(status),
		},
	}); err != nil {
		return domain.Reward{}, err
	}
	if err := s.enqueueDomainRewardCalculated(ctx, reward); err != nil {
		return domain.Reward{}, err
	}
	if status == domain.RewardStatusEligible {
		if err := s.enqueueDomainRewardPayoutEligible(ctx, reward); err != nil {
			return domain.Reward{}, err
		}
	}
	if err := s.FlushOutbox(ctx); err != nil {
		return domain.Reward{}, err
	}
	payload, err := json.Marshal(reward)
	if err != nil {
		return domain.Reward{}, err
	}
	if err := s.idempotency.Complete(ctx, idempotencyKey, 201, payload, s.nowFn()); err != nil {
		return domain.Reward{}, err
	}
	return reward, nil
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
