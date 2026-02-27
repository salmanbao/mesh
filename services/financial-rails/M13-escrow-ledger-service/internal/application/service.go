package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/domain"
)

func (s *Service) CreateHold(ctx context.Context, actor Actor, input CreateHoldInput) (domain.EscrowHold, error) {
	if strings.TrimSpace(actor.SubjectID) == "" { return domain.EscrowHold{}, domain.ErrUnauthorized }
	if strings.TrimSpace(actor.IdempotencyKey) == "" { return domain.EscrowHold{}, domain.ErrIdempotencyRequired }
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.CreatorID = strings.TrimSpace(input.CreatorID)
	if input.CampaignID == "" || input.CreatorID == "" || input.Amount <= 0 { return domain.EscrowHold{}, domain.ErrInvalidInput }
	requestHash := hashJSON(input)
	if cached, ok, err := s.getIdempotentHold(ctx, actor.IdempotencyKey, requestHash); err != nil { return domain.EscrowHold{}, err } else if ok { return cached, nil }
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil { return domain.EscrowHold{}, err }
	now := s.nowFn()
	hold := domain.EscrowHold{EscrowID: uuid.NewString(), CampaignID: input.CampaignID, CreatorID: input.CreatorID, OriginalAmount: input.Amount, RemainingAmount: input.Amount, Status: domain.HoldStatusActive, HeldAt: now, UpdatedAt: now}
	if err := s.holds.Create(ctx, hold); err != nil { return domain.EscrowHold{}, err }
	if err := s.ledger.Append(ctx, domain.LedgerEntry{EntryID: uuid.NewString(), EscrowID: hold.EscrowID, CampaignID: hold.CampaignID, EntryType: "hold", Amount: hold.OriginalAmount, OccurredAt: now}); err != nil { return domain.EscrowHold{}, err }
	if err := s.enqueueHoldCreated(ctx, hold, actor.RequestID, now); err != nil { return domain.EscrowHold{}, err }
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, hold)
	return hold, nil
}

func (s *Service) Release(ctx context.Context, actor Actor, input ReleaseInput) (domain.EscrowHold, error) {
	if strings.TrimSpace(actor.SubjectID) == "" { return domain.EscrowHold{}, domain.ErrUnauthorized }
	if strings.TrimSpace(actor.IdempotencyKey) == "" { return domain.EscrowHold{}, domain.ErrIdempotencyRequired }
	input.EscrowID = strings.TrimSpace(input.EscrowID)
	if input.EscrowID == "" || input.Amount <= 0 { return domain.EscrowHold{}, domain.ErrInvalidInput }
	requestHash := hashJSON(input)
	if cached, ok, err := s.getIdempotentHold(ctx, actor.IdempotencyKey, requestHash); err != nil { return domain.EscrowHold{}, err } else if ok { return cached, nil }
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil { return domain.EscrowHold{}, err }
	hold, err := s.holds.GetByID(ctx, input.EscrowID)
	if err != nil { return domain.EscrowHold{}, err }
	if hold.Status == domain.HoldStatusRefunded || hold.Status == domain.HoldStatusFullyReleased { return domain.EscrowHold{}, domain.ErrHoldClosed }
	if input.Amount > hold.RemainingAmount { return domain.EscrowHold{}, domain.ErrInsufficientEscrow }
	now := s.nowFn()
	hold.ReleasedAmount += input.Amount
	hold.RemainingAmount -= input.Amount
	if hold.RemainingAmount == 0 { hold.Status = domain.HoldStatusFullyReleased } else { hold.Status = domain.HoldStatusPartialRelease }
	hold.UpdatedAt = now
	if err := s.holds.Update(ctx, hold); err != nil { return domain.EscrowHold{}, err }
	if err := s.ledger.Append(ctx, domain.LedgerEntry{EntryID: uuid.NewString(), EscrowID: hold.EscrowID, CampaignID: hold.CampaignID, EntryType: "release", Amount: input.Amount, OccurredAt: now}); err != nil { return domain.EscrowHold{}, err }
	if hold.Status == domain.HoldStatusFullyReleased {
		if err := s.enqueueHoldFullyReleased(ctx, hold.EscrowID, actor.RequestID, now); err != nil { return domain.EscrowHold{}, err }
	} else {
		if err := s.enqueuePartialRelease(ctx, hold.EscrowID, input.Amount, hold.RemainingAmount, actor.RequestID, now); err != nil { return domain.EscrowHold{}, err }
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, hold)
	return hold, nil
}

func (s *Service) Refund(ctx context.Context, actor Actor, input RefundInput) (domain.EscrowHold, error) {
	if strings.TrimSpace(actor.SubjectID) == "" { return domain.EscrowHold{}, domain.ErrUnauthorized }
	if strings.TrimSpace(actor.IdempotencyKey) == "" { return domain.EscrowHold{}, domain.ErrIdempotencyRequired }
	input.EscrowID = strings.TrimSpace(input.EscrowID)
	if input.EscrowID == "" { return domain.EscrowHold{}, domain.ErrInvalidInput }
	requestHash := hashJSON(input)
	if cached, ok, err := s.getIdempotentHold(ctx, actor.IdempotencyKey, requestHash); err != nil { return domain.EscrowHold{}, err } else if ok { return cached, nil }
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil { return domain.EscrowHold{}, err }
	hold, err := s.holds.GetByID(ctx, input.EscrowID)
	if err != nil { return domain.EscrowHold{}, err }
	if hold.Status == domain.HoldStatusRefunded || hold.Status == domain.HoldStatusFullyReleased { return domain.EscrowHold{}, domain.ErrHoldClosed }
	amount := hold.RemainingAmount
	if input.Amount != nil { amount = *input.Amount }
	if amount <= 0 || amount > hold.RemainingAmount { return domain.EscrowHold{}, domain.ErrInsufficientEscrow }
	now := s.nowFn()
	hold.RefundedAmount += amount
	hold.RemainingAmount -= amount
	if hold.RemainingAmount == 0 { hold.Status = domain.HoldStatusRefunded } else { hold.Status = domain.HoldStatusPartialRelease }
	hold.UpdatedAt = now
	if err := s.holds.Update(ctx, hold); err != nil { return domain.EscrowHold{}, err }
	if err := s.ledger.Append(ctx, domain.LedgerEntry{EntryID: uuid.NewString(), EscrowID: hold.EscrowID, CampaignID: hold.CampaignID, EntryType: "refund", Amount: amount, OccurredAt: now}); err != nil { return domain.EscrowHold{}, err }
	if err := s.enqueueRefundProcessed(ctx, hold.EscrowID, amount, actor.RequestID, now); err != nil { return domain.EscrowHold{}, err }
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, hold)
	return hold, nil
}

func (s *Service) GetWalletBalance(ctx context.Context, actor Actor, campaignID string) (domain.WalletBalance, error) {
	if strings.TrimSpace(actor.SubjectID) == "" { return domain.WalletBalance{}, domain.ErrUnauthorized }
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" { return domain.WalletBalance{}, domain.ErrInvalidInput }
	entries, err := s.ledger.ListByCampaignID(ctx, campaignID)
	if err != nil { return domain.WalletBalance{}, err }
	out := domain.WalletBalance{CampaignID: campaignID, CalculatedAt: s.nowFn()}
	for _, e := range entries {
		switch e.EntryType {
		case "hold":
			out.HeldBalance += e.Amount
		case "release":
			out.ReleasedBalance += e.Amount
		case "refund":
			out.RefundedBalance += e.Amount
		}
	}
	out.NetEscrowBalance = out.HeldBalance - out.ReleasedBalance - out.RefundedBalance
	if out.NetEscrowBalance < 0 { out.NetEscrowBalance = 0 }
	return out, nil
}

func (s *Service) getIdempotentHold(ctx context.Context, key, requestHash string) (domain.EscrowHold, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" { return domain.EscrowHold{}, false, nil }
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil { return domain.EscrowHold{}, false, err }
	if rec.RequestHash != requestHash { return domain.EscrowHold{}, false, domain.ErrIdempotencyConflict }
	if len(rec.ResponseBody) == 0 { return domain.EscrowHold{}, false, nil }
	var out domain.EscrowHold
	if err := json.Unmarshal(rec.ResponseBody, &out); err != nil { return domain.EscrowHold{}, false, nil }
	return out, true, nil
}
func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil { return nil }
	err := s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
	if err == domain.ErrConflict { return domain.ErrIdempotencyConflict }
	return err
}
func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" { return nil }
	b, _ := json.Marshal(payload)
	return s.idempotency.Complete(ctx, key, code, b, s.nowFn())
}
func hashJSON(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
