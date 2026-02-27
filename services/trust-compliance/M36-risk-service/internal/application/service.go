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
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/contracts"
	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/domain"
)

func (s *Service) GetSellerRiskDashboard(ctx context.Context, actor Actor) (domain.RiskDashboard, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.RiskDashboard{}, domain.ErrUnauthorized
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if role == "" {
		role = "seller"
	}
	sellerID := actor.SubjectID

	authSummary, _ := s.auth.GetAuthSummary(ctx, sellerID)
	profileSummary, _ := s.profile.GetProfileSummary(ctx, sellerID)
	fraudSummary, _ := s.fraud.GetFraudSummary(ctx, sellerID)
	moderationSummary, _ := s.moderation.GetModerationSummary(ctx, sellerID)
	resolutionSummary, _ := s.resolution.GetResolutionSummary(ctx, sellerID)
	reputationSummary, _ := s.reputation.GetReputationSummary(ctx, sellerID)
	_ = reputationSummary

	profile, err := s.riskProfiles.GetBySellerID(ctx, sellerID)
	if err != nil {
		score := domain.ScoreFromSignals(resolutionSummary.DisputeRate, authSummary.AccountAgeDays, fraudSummary.SalesVelocity, moderationSummary.ProductClarityScore, fraudSummary.FraudHistoryCount)
		now := s.nowFn()
		profile = domain.SellerRiskProfile{
			SellerID: sellerID, CurrentRiskScore: score, PreviousRiskScore: score, RiskLevel: domain.RiskLevel(score),
			DisputeRate: resolutionSummary.DisputeRate, AccountAgeDays: authSummary.AccountAgeDays, SalesVelocity: fraudSummary.SalesVelocity,
			ProductClarityScore: moderationSummary.ProductClarityScore, FraudHistoryCount: fraudSummary.FraudHistoryCount,
			ReservePercentage: domain.ReservePercentageForScore(score), UpdatedAt: now,
		}
		if s.riskProfiles != nil {
			_ = s.riskProfiles.Upsert(ctx, profile)
		}
	}

	escrow, err := s.escrow.GetBySellerID(ctx, sellerID)
	if err != nil {
		now := s.nowFn()
		next := now.AddDate(0, 0, 7)
		escrow = domain.SellerEscrow{EscrowID: uuid.NewString(), SellerID: sellerID, EscrowedAmount: 3500, AvailableBalance: profileSummary.AvailableBalance, ReservePercentage: profile.ReservePercentage, NextReleaseMilestone: &next, UpdatedAt: now}
		if s.escrow != nil {
			_ = s.escrow.Upsert(ctx, escrow)
		}
	}

	triggers, _ := s.reserveLogs.ListBySeller(ctx, sellerID, 5)
	flags, _ := s.fraudFlags.ListBySeller(ctx, sellerID, 5)
	alerts := []string{}
	if resolutionSummary.DisputeRate >= 0.018 {
		alerts = append(alerts, "Dispute rate at 1.8%; monitor to avoid 2% threshold that triggers additional reserve")
	}
	if profile.CurrentRiskScore >= 0.85 {
		alerts = append(alerts, "Critical risk score: account may be subject to suspension")
	}

	schedule := []domain.ReleaseScheduleItem{}
	if escrow.NextReleaseMilestone != nil && escrow.EscrowedAmount > 0 {
		weekly := escrow.EscrowedAmount * 0.10
		for i := 0; i < 2; i++ {
			schedule = append(schedule, domain.ReleaseScheduleItem{Date: escrow.NextReleaseMilestone.AddDate(0, 0, 7*i).Format("2006-01-02"), Amount: weekly, Status: "scheduled"})
		}
	}
	recentFlags := make([]domain.SellerFlag, 0, len(triggers))
	for _, t := range triggers {
		recentFlags = append(recentFlags, domain.SellerFlag{Reason: t.Reason, Date: t.CreatedAt.Format("2006-01-02"), ActionTaken: fmt.Sprintf("Reserve set to %d%%", t.AppliedReservePct), Status: "active"})
	}
	fraudAlerts := make([]domain.FraudAlert, 0, len(flags))
	for _, f := range flags {
		fraudAlerts = append(fraudAlerts, domain.FraudAlert{FlagID: f.FlagID, PatternType: f.PatternType, ConfidenceScore: f.ConfidenceScore, RecommendedAction: f.RecommendedAction, CreatedAt: f.CreatedAt.Format(time.RFC3339)})
	}

	if role == "seller" && sellerID != actor.SubjectID {
		return domain.RiskDashboard{}, domain.ErrForbidden
	}

	return domain.RiskDashboard{
		CurrentRiskScore: profile.CurrentRiskScore,
		RiskLevel:        profile.RiskLevel,
		ReserveStatus:    domain.ReserveStatus{PercentageHeld: escrow.ReservePercentage, EscrowedAmount: escrow.EscrowedAmount, AvailableBalance: escrow.AvailableBalance, NextReleaseMilestone: dateOrEmpty(escrow.NextReleaseMilestone), EstimatedReleaseSchedule: schedule},
		DisputeHistory:   domain.DisputeHistory{TotalDisputes12M: resolutionSummary.TotalDisputes12M, Breakdown: domain.DisputeHistoryBreakdown{ResolvedForSeller: resolutionSummary.ResolvedForSeller, ResolvedForBuyer: resolutionSummary.ResolvedForBuyer, PartialRefund: resolutionSummary.PartialRefund}, LastDisputeDate: resolutionSummary.LastDisputeDate, LastDisputeOutcome: resolutionSummary.LastOutcome},
		RecentFlags:      recentFlags,
		FraudAlerts:      fraudAlerts,
		Alerts:           alerts,
	}, nil
}

func (s *Service) FileDispute(ctx context.Context, actor Actor, input FileDisputeInput) (domain.DisputeLog, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DisputeLog{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DisputeLog{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.TransactionID) == "" || strings.TrimSpace(input.Reason) == "" || strings.TrimSpace(input.BuyerClaim) == "" {
		return domain.DisputeLog{}, domain.ErrInvalidInput
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentDispute(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DisputeLog{}, err
	} else if ok {
		return cached, nil
	}
	if existing, err := s.disputes.GetByTransactionID(ctx, strings.TrimSpace(input.TransactionID)); err == nil && existing.DisputeID != "" {
		return domain.DisputeLog{}, domain.ErrConflict
	}
	now := s.nowFn()
	deadline := now.Add(48 * time.Hour)
	sellerID := deriveSellerID(input.TransactionID)
	row := domain.DisputeLog{DisputeID: uuid.NewString(), TransactionID: strings.TrimSpace(input.TransactionID), SellerID: sellerID, BuyerID: actor.SubjectID, DisputeType: domain.NormalizeDisputeType(input.DisputeType), Reason: strings.TrimSpace(input.Reason), BuyerClaim: strings.TrimSpace(input.BuyerClaim), Status: "pending", SellerResponseDeadline: &deadline, FiledAt: now, UpdatedAt: now}
	if err := s.disputes.Create(ctx, row); err != nil {
		return domain.DisputeLog{}, err
	}
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 201, row); err != nil {
		return domain.DisputeLog{}, err
	}
	return row, nil
}

func (s *Service) SubmitDisputeEvidence(ctx context.Context, actor Actor, disputeID string, input SubmitEvidenceInput) (domain.DisputeEvidence, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DisputeEvidence{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DisputeEvidence{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(disputeID) == "" || strings.TrimSpace(input.Filename) == "" {
		return domain.DisputeEvidence{}, domain.ErrInvalidInput
	}
	dispute, err := s.disputes.GetByID(ctx, strings.TrimSpace(disputeID))
	if err != nil {
		return domain.DisputeEvidence{}, err
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	if role != "seller" && role != "admin" && role != "moderator" && role != "finance" {
		return domain.DisputeEvidence{}, domain.ErrForbidden
	}
	if role == "seller" && dispute.SellerID != actor.SubjectID {
		return domain.DisputeEvidence{}, domain.ErrForbidden
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentEvidence(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DisputeEvidence{}, err
	} else if ok {
		return cached, nil
	}
	now := s.nowFn()
	row := domain.DisputeEvidence{EvidenceID: uuid.NewString(), DisputeID: dispute.DisputeID, SellerID: dispute.SellerID, Filename: strings.TrimSpace(input.Filename), Description: strings.TrimSpace(input.Description), FileURL: strings.TrimSpace(input.FileURL), SizeBytes: input.SizeBytes, MimeType: strings.TrimSpace(input.MimeType), UploadedAt: now}
	if err := s.evidence.Create(ctx, row); err != nil {
		return domain.DisputeEvidence{}, err
	}
	dispute.EvidenceCount++
	dispute.Status = "manual_review"
	dispute.UpdatedAt = now
	_ = s.disputes.Update(ctx, dispute)
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 200, row); err != nil {
		return domain.DisputeEvidence{}, err
	}
	return row, nil
}

func (s *Service) HandleChargebackWebhook(ctx context.Context, bearerToken string, input ChargebackInput) (map[string]any, error) {
	if strings.TrimSpace(bearerToken) != s.cfg.WebhookBearerToken {
		return nil, domain.ErrUnauthorized
	}
	if strings.TrimSpace(input.EventID) == "" || strings.TrimSpace(input.ChargeID) == "" || strings.TrimSpace(input.SellerID) == "" {
		return nil, domain.ErrInvalidInput
	}
	if strings.TrimSpace(input.EventType) == "" || strings.TrimSpace(input.SourceService) == "" || strings.TrimSpace(input.TraceID) == "" || strings.TrimSpace(input.SchemaVersion) == "" {
		return nil, domain.ErrInvalidEnvelope
	}
	payloadBytes, _ := json.Marshal(map[string]any{"seller_id": strings.TrimSpace(input.SellerID)})
	if err := validatePartitionKeyInvariant(contracts.EventEnvelope{
		PartitionKeyPath: strings.TrimSpace(input.PartitionKeyPath),
		PartitionKey:     strings.TrimSpace(input.PartitionKey),
		Data:             payloadBytes,
	}, "data.seller_id"); err != nil {
		return nil, err
	}
	now := s.nowFn()
	if s.eventDedup != nil {
		dup, err := s.eventDedup.IsDuplicate(ctx, input.EventID, now)
		if err != nil {
			return nil, err
		}
		if dup {
			return map[string]any{"accepted": true, "duplicate": true}, nil
		}
	}
	occurredAt, err := parseRFC3339OrNow(input.OccurredAt, now)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	_ = occurredAt
	escrow, err := s.escrow.GetBySellerID(ctx, input.SellerID)
	if err != nil {
		escrow = domain.SellerEscrow{EscrowID: uuid.NewString(), SellerID: input.SellerID, EscrowedAmount: 0, AvailableBalance: 0, ReservePercentage: 0, UpdatedAt: now}
	}
	fee := 25.0
	total := input.Amount + fee
	deductedFrom := "available"
	negative := false
	if escrow.AvailableBalance >= total {
		escrow.AvailableBalance -= total
	} else if escrow.EscrowedAmount >= total {
		deductedFrom = "escrow"
		escrow.EscrowedAmount -= total
	} else {
		deductedFrom = "mixed"
		remaining := total
		if escrow.AvailableBalance > 0 {
			remaining -= escrow.AvailableBalance
			escrow.AvailableBalance = 0
		}
		if escrow.EscrowedAmount > 0 {
			remaining -= escrow.EscrowedAmount
			escrow.EscrowedAmount = 0
		}
		negative = remaining > 0
	}
	escrow.UpdatedAt = now
	_ = s.escrow.Upsert(ctx, escrow)
	debt := domain.SellerDebtLog{DebtID: uuid.NewString(), SellerID: input.SellerID, Reason: "chargeback", ChargeID: input.ChargeID, Amount: input.Amount, FeeAmount: fee, TotalDeducted: total, DeductedFrom: deductedFrom, NegativeBalance: negative, CreatedAt: now}
	_ = s.debtLogs.Create(ctx, debt)
	trig := domain.ReserveTriggerLog{TriggerID: uuid.NewString(), SellerID: input.SellerID, TriggerType: "chargeback", Reason: nonEmpty(input.DisputeReason, "chargeback"), AppliedReservePct: maxInt(escrow.ReservePercentage, 50), ReserveChangeAmount: total, CreatedAt: now}
	_ = s.reserveLogs.Create(ctx, trig)
	if s.eventDedup != nil {
		_ = s.eventDedup.MarkProcessed(ctx, input.EventID, input.EventType, now.Add(s.cfg.EventDedupTTL))
	}
	return map[string]any{"accepted": true, "charge_id": input.ChargeID, "seller_id": input.SellerID, "processed_at": now}, nil
}

func (s *Service) getIdempotentDispute(ctx context.Context, key, requestHash string) (domain.DisputeLog, bool, error) {
	if s.idempotency == nil {
		return domain.DisputeLog{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, key, now)
	if err != nil {
		return domain.DisputeLog{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.DisputeLog{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.DisputeLog
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.DisputeLog{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, key, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.DisputeLog{}, false, err
	}
	return domain.DisputeLog{}, false, nil
}

func (s *Service) getIdempotentEvidence(ctx context.Context, key, requestHash string) (domain.DisputeEvidence, bool, error) {
	if s.idempotency == nil {
		return domain.DisputeEvidence{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, key, now)
	if err != nil {
		return domain.DisputeEvidence{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domain.DisputeEvidence{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.DisputeEvidence
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.DisputeEvidence{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, key, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.DisputeEvidence{}, false, err
	}
	return domain.DisputeEvidence{}, false, nil
}

func (s *Service) completeIdempotent(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.idempotency.Complete(ctx, key, code, b, s.nowFn())
}

func hashPayload(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func dateOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func nonEmpty(v, f string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return f
}
func deriveSellerID(transactionID string) string {
	if transactionID == "" {
		return "seller_unknown"
	}
	return "seller_" + strings.TrimPrefix(transactionID, "txn_")
}
func parseRFC3339OrNow(raw string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
