package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/trust-compliance/M96-referral-fraud-detection-service/internal/domain"
)

func (s *Service) ScoreReferral(ctx context.Context, actor Actor, input ScoreInput) (domain.FraudDecision, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.FraudDecision{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.FraudDecision{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.EventID) == "" || strings.TrimSpace(input.AffiliateID) == "" {
		return domain.FraudDecision{}, domain.ErrInvalidInput
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentDecision(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.FraudDecision{}, err
	} else if ok {
		return cached, nil
	}
	dec, err := s.scoreAndPersist(ctx, input, actor.RequestID, true)
	if err != nil {
		return domain.FraudDecision{}, err
	}
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 200, dec); err != nil {
		return domain.FraudDecision{}, err
	}
	return dec, nil
}

func (s *Service) GetDecisionByEventID(ctx context.Context, actor Actor, eventID string) (domain.FraudDecision, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.FraudDecision{}, domain.ErrUnauthorized
	}
	if normalizeRole(actor.Role) != "admin" && normalizeRole(actor.Role) != "analyst" {
		return domain.FraudDecision{}, domain.ErrForbidden
	}
	return s.decisions.GetByEventID(ctx, strings.TrimSpace(eventID))
}

func (s *Service) SubmitDispute(ctx context.Context, actor Actor, input SubmitDisputeInput) (domain.DisputeCase, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DisputeCase{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DisputeCase{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(input.DecisionID) == "" || strings.TrimSpace(input.SubmittedBy) == "" || strings.TrimSpace(input.EvidenceURL) == "" {
		return domain.DisputeCase{}, domain.ErrInvalidInput
	}
	if _, err := url.ParseRequestURI(strings.TrimSpace(input.EvidenceURL)); err != nil {
		return domain.DisputeCase{}, domain.ErrInvalidInput
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentDispute(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DisputeCase{}, err
	} else if ok {
		return cached, nil
	}
	decision, err := s.decisions.GetByDecisionID(ctx, strings.TrimSpace(input.DecisionID))
	if err != nil {
		return domain.DisputeCase{}, err
	}
	if existing, err := s.disputes.GetByDecisionID(ctx, decision.DecisionID); err == nil && existing.DisputeID != "" {
		return domain.DisputeCase{}, domain.ErrConflict
	}
	now := s.nowFn()
	dispute := domain.DisputeCase{DisputeID: uuid.NewString(), DecisionID: decision.DecisionID, EventID: decision.EventID, SubmittedBy: strings.TrimSpace(input.SubmittedBy), EvidenceURL: strings.TrimSpace(input.EvidenceURL), Status: "submitted", CreatedAt: now}
	if err := s.disputes.Create(ctx, dispute); err != nil {
		return domain.DisputeCase{}, err
	}
	_ = s.auditLogs.Create(ctx, domain.AuditLog{AuditID: uuid.NewString(), EntityType: "dispute_case", EntityID: dispute.DisputeID, Action: "dispute_submitted", Summary: "fraud decision dispute submitted", TraceID: actor.RequestID, CreatedAt: now})
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 201, dispute); err != nil {
		return domain.DisputeCase{}, err
	}
	return dispute, nil
}

func (s *Service) GetMetrics(ctx context.Context, actor Actor) (domain.MetricsSnapshot, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.MetricsSnapshot{}, domain.ErrUnauthorized
	}
	if normalizeRole(actor.Role) != "admin" && normalizeRole(actor.Role) != "analyst" {
		return domain.MetricsSnapshot{}, domain.ErrForbidden
	}
	recent, _ := s.decisions.ListRecent(ctx, 500)
	active, _ := s.disputes.ListByStatus(ctx, "submitted", 500)
	fraudCount := 0
	protected := 0.0
	vectorCounts := map[string]int{}
	for _, d := range recent {
		if d.Decision == "block" || d.Decision == "flag" {
			fraudCount++
		}
		if d.Decision == "block" {
			protected += 50
		} // synthetic estimate for in-memory impl
		for _, f := range d.Flags {
			vectorCounts[f]++
		}
	}
	attack := make([]domain.AttackVector, 0, len(vectorCounts))
	for k, v := range vectorCounts {
		pct := 0.0
		if len(recent) > 0 {
			pct = float64(v) * 100 / float64(len(recent))
		}
		attack = append(attack, domain.AttackVector{Type: k, Percent: pct})
	}
	return domain.MetricsSnapshot{
		FraudRate:            safeRateInt(fraudCount, len(recent)),
		AttackVectors:        attack,
		RevenueProtectedUSD:  protected,
		ActiveInvestigations: len(active),
		AppealQueue:          len(active),
		GeneratedAt:          s.nowFn(),
	}, nil
}

func (s *Service) scoreAndPersist(ctx context.Context, input ScoreInput, traceID string, allowIdempotentConflict bool) (domain.FraudDecision, error) {
	now := s.nowFn()
	if input.EventType == "" {
		input.EventType = domain.EventAffiliateClickTracked
	}
	if !isValidInputEventTypeForManual(input.EventType) {
		return domain.FraudDecision{}, domain.ErrInvalidInput
	}
	if strings.TrimSpace(input.OccurredAt) == "" {
		input.OccurredAt = now.Format(time.RFC3339)
	}
	occurredAt, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil {
		occurredAt = now
	}
	_, _ = s.affiliate.GetAffiliateSummary(ctx, input.AffiliateID)
	if existing, err := s.decisions.GetByEventID(ctx, input.EventID); err == nil && existing.DecisionID != "" {
		if allowIdempotentConflict {
			return existing, nil
		}
		return domain.FraudDecision{}, domain.ErrConflict
	}
	refEvent := domain.ReferralEvent{
		EventID: input.EventID, SourceEventType: input.EventType, AffiliateID: input.AffiliateID, ReferralToken: strings.TrimSpace(input.ReferralToken), ReferrerID: strings.TrimSpace(input.ReferrerID), UserID: strings.TrimSpace(input.UserID), ConversionID: strings.TrimSpace(input.ConversionID), OrderID: strings.TrimSpace(input.OrderID), TransactionID: strings.TrimSpace(input.OrderID), Amount: input.Amount, ClickIP: strings.TrimSpace(input.ClickIP), UserAgent: strings.TrimSpace(input.UserAgent), DeviceFingerprintHash: strings.TrimSpace(input.DeviceFingerprintHash), FormFillTimeMS: input.FormFillTimeMS, MouseMovementCount: input.MouseMovementCount, KeyboardCPS: input.KeyboardCPS, Country: strings.TrimSpace(input.Region), Metadata: cloneMetadata(input.Metadata), OccurredAt: occurredAt.UTC(), CreatedAt: now, RawPayload: append([]byte(nil), input.RawPayload...),
	}
	if refEvent.ClickIP != "" && !validIPOrHash(refEvent.ClickIP) {
		return domain.FraudDecision{}, domain.ErrInvalidInput
	}
	if err := s.referralEvents.Create(ctx, refEvent); err != nil {
		return domain.FraudDecision{}, err
	}
	var fp *domain.DeviceFingerprint
	if refEvent.DeviceFingerprintHash != "" {
		fpr, err := s.fingerprints.UpsertSeen(ctx, refEvent.DeviceFingerprintHash, refEvent.ClickIP, now)
		if err != nil {
			return domain.FraudDecision{}, err
		}
		fp = &fpr
	}
	clusterSize := 0
	clusterID := ""
	if refEvent.ClickIP != "" {
		cl, err := s.clusters.UpsertByKey(ctx, refEvent.ClickIP, "shared_ip", now)
		if err != nil {
			return domain.FraudDecision{}, err
		}
		clusterSize = cl.Size
		if cl.Size >= 3 {
			clusterID = cl.ClusterID
		}
	}
	threshold, policyVersion := s.resolveThreshold(ctx, refEvent.AffiliateID, refEvent.Country, valueOrDefault(refEvent.Metadata["campaign_type"], input.CampaignType))
	score, flags := domain.ScoreReferral(domain.ScoreRequest{EventID: refEvent.EventID, EventType: refEvent.SourceEventType, AffiliateID: refEvent.AffiliateID, ReferralToken: refEvent.ReferralToken, ReferrerID: refEvent.ReferrerID, UserID: refEvent.UserID, Amount: refEvent.Amount, ClickIP: refEvent.ClickIP, UserAgent: refEvent.UserAgent, DeviceFingerprintHash: refEvent.DeviceFingerprintHash, FormFillTimeMS: refEvent.FormFillTimeMS, MouseMovementCount: refEvent.MouseMovementCount, KeyboardCPS: refEvent.KeyboardCPS}, fp, clusterSize)
	decisionValue, tier := domain.NormalizeDecision(score, threshold, flags)
	decision := domain.FraudDecision{DecisionID: uuid.NewString(), EventID: refEvent.EventID, RiskScore: score, Decision: decisionValue, RiskTier: tier, Flags: flags, ModelVersion: s.cfg.ModelVersion, PolicyVersion: policyVersion, ClusterID: clusterID, CreatedAt: now, SourceEventType: refEvent.SourceEventType}
	if err := s.decisions.Create(ctx, decision); err != nil {
		return domain.FraudDecision{}, err
	}
	_ = s.auditLogs.Create(ctx, domain.AuditLog{AuditID: uuid.NewString(), EntityType: "fraud_decision", EntityID: decision.DecisionID, Action: "decision_generated", Summary: fmt.Sprintf("decision=%s risk_score=%.3f", decision.Decision, decision.RiskScore), TraceID: strings.TrimSpace(traceID), CreatedAt: now})
	return decision, nil
}

func (s *Service) resolveThreshold(ctx context.Context, affiliateID, region, campaignType string) (float64, string) {
	threshold := s.cfg.DefaultThreshold
	version := s.cfg.PolicyVersion
	policies, err := s.policies.ListActive(ctx)
	if err != nil {
		return threshold, version
	}
	for _, p := range policies {
		if p.AffiliateID != "" && p.AffiliateID != affiliateID {
			continue
		}
		if p.Region != "" && !strings.EqualFold(p.Region, region) {
			continue
		}
		if p.CampaignType != "" && !strings.EqualFold(p.CampaignType, campaignType) {
			continue
		}
		threshold = p.Threshold
		if p.Version != "" {
			version = p.Version
		}
		break
	}
	return threshold, version
}

func cloneMetadata(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}
func valueOrDefault(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}
func validIPOrHash(v string) bool {
	if strings.Contains(v, ".") || strings.Contains(v, ":") {
		return true
	}
	return len(strings.TrimSpace(v)) >= 8
}
func safeRateInt(n, d int) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / float64(d)
}
func isValidInputEventTypeForManual(et string) bool {
	return et == domain.EventAffiliateClickTracked || et == domain.EventAffiliateAttributionCreate || et == domain.EventTransactionSucceeded || et == domain.EventUserRegistered
}

func hashPayload(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) getIdempotentDecision(ctx context.Context, key, requestHash string) (domain.FraudDecision, bool, error) {
	if s.idempotency == nil {
		return domain.FraudDecision{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, key, now)
	if err != nil {
		return domain.FraudDecision{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, key, "")
			return domain.FraudDecision{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.FraudDecision
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.FraudDecision{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, key, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.FraudDecision{}, false, err
	}
	return domain.FraudDecision{}, false, nil
}

func (s *Service) getIdempotentDispute(ctx context.Context, key, requestHash string) (domain.DisputeCase, bool, error) {
	if s.idempotency == nil {
		return domain.DisputeCase{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, key, now)
	if err != nil {
		return domain.DisputeCase{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, key, "")
			return domain.DisputeCase{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.DisputeCase
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.DisputeCase{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, key, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.DisputeCase{}, false, err
	}
	return domain.DisputeCase{}, false, nil
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
