package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) GetConsent(ctx context.Context, actor Actor, userID string) (domain.ConsentRecord, error) {
	userID, err := s.resolveUserForView(actor, userID)
	if err != nil {
		return domain.ConsentRecord{}, err
	}
	return s.consents.GetByUserID(ctx, userID)
}

func (s *Service) UpdateConsent(ctx context.Context, actor Actor, in UpdateConsentInput) (domain.ConsentRecord, error) {
	userID, err := s.resolveUserForOperate(actor, in.UserID)
	if err != nil {
		return domain.ConsentRecord{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ConsentRecord{}, domain.ErrIdempotencyRequired
	}
	if len(in.Preferences) == 0 {
		return domain.ConsentRecord{}, domain.ErrInvalidInput
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return domain.ConsentRecord{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{
		"op":          "update_consent",
		"user_id":     userID,
		"preferences": in.Preferences,
		"reason":      reason,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ConsentRecord{}, err
	} else if ok {
		var out domain.ConsentRecord
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ConsentRecord{}, err
	}

	row := domain.ConsentRecord{
		UserID:      userID,
		Preferences: cloneBoolMap(in.Preferences),
		Status:      domain.ConsentStatusActive,
		UpdatedAt:   s.nowFn(),
		UpdatedBy:   actor.SubjectID,
		LastReason:  reason,
	}
	if err := s.consents.Upsert(ctx, row); err != nil {
		return domain.ConsentRecord{}, err
	}
	_ = s.consents.AppendHistory(ctx, domain.ConsentHistory{
		EventID:    nextID("hist"),
		EventType:  "consent.updated",
		UserID:     userID,
		Reason:     reason,
		ChangedBy:  actor.SubjectID,
		OccurredAt: row.UpdatedAt,
	})
	s.appendAudit(ctx, "consent.updated", actor.SubjectID, userID, map[string]string{"reason": reason})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) WithdrawConsent(ctx context.Context, actor Actor, in WithdrawConsentInput) (domain.ConsentRecord, error) {
	userID, err := s.resolveUserForOperate(actor, in.UserID)
	if err != nil {
		return domain.ConsentRecord{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ConsentRecord{}, domain.ErrIdempotencyRequired
	}
	category := strings.ToLower(strings.TrimSpace(in.Category))
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return domain.ConsentRecord{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{
		"op":       "withdraw_consent",
		"user_id":  userID,
		"category": category,
		"reason":   reason,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ConsentRecord{}, err
	} else if ok {
		var out domain.ConsentRecord
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ConsentRecord{}, err
	}

	row, err := s.consents.GetByUserID(ctx, userID)
	if err != nil {
		if err != domain.ErrNotFound {
			return domain.ConsentRecord{}, err
		}
		row = domain.ConsentRecord{
			UserID:      userID,
			Preferences: map[string]bool{},
			Status:      domain.ConsentStatusNoConsent,
		}
	}
	if row.Preferences == nil {
		row.Preferences = map[string]bool{}
	}
	if category == "" || category == "all" {
		for key := range row.Preferences {
			row.Preferences[key] = false
		}
		row.Status = domain.ConsentStatusWithdrawn
		category = "all"
	} else {
		row.Preferences[category] = false
		row.Status = domain.ConsentStatusActive
	}
	row.UpdatedAt = s.nowFn()
	row.UpdatedBy = actor.SubjectID
	row.LastReason = reason
	if err := s.consents.Upsert(ctx, row); err != nil {
		return domain.ConsentRecord{}, err
	}
	_ = s.consents.AppendHistory(ctx, domain.ConsentHistory{
		EventID:    nextID("hist"),
		EventType:  "consent.withdrawn",
		UserID:     userID,
		Category:   category,
		Reason:     reason,
		ChangedBy:  actor.SubjectID,
		OccurredAt: row.UpdatedAt,
	})
	s.appendAudit(ctx, "consent.withdrawn", actor.SubjectID, userID, map[string]string{"category": category, "reason": reason})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) ListHistory(ctx context.Context, actor Actor, userID string, limit int) ([]domain.ConsentHistory, error) {
	userID, err := s.resolveUserForView(actor, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	return s.consents.ListHistory(ctx, userID, limit)
}

func canOperate(actor Actor, userID string) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	actorID := strings.TrimSpace(actor.SubjectID)
	userID = strings.TrimSpace(userID)
	if actorID == "" || userID == "" {
		return false
	}
	return actorID == userID || role == "admin" || role == "compliance" || role == "legal"
}

func canView(actor Actor, userID string) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	actorID := strings.TrimSpace(actor.SubjectID)
	userID = strings.TrimSpace(userID)
	if actorID == "" || userID == "" {
		return false
	}
	return actorID == userID || role == "admin" || role == "compliance" || role == "legal" || role == "support"
}

func (s *Service) resolveUserForView(actor Actor, requested string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if canView(actor, requested) {
		return requested, nil
	}
	return "", domain.ErrForbidden
}

func (s *Service) resolveUserForOperate(actor Actor, requested string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if canOperate(actor, requested) {
		return requested, nil
	}
	return "", domain.ErrForbidden
}

func (s *Service) appendAudit(ctx context.Context, eventType, actorID, userID string, metadata map[string]string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Append(ctx, domain.AuditLog{
		EventID:    nextID("audit"),
		EventType:  eventType,
		UserID:     userID,
		ActorID:    actorID,
		OccurredAt: s.nowFn(),
		Metadata:   metadata,
	})
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
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
