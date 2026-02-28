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

	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) ListPolicies(ctx context.Context, actor Actor) ([]domain.RetentionPolicy, error) {
	if !canView(actor) {
		return nil, authorizeError(actor)
	}
	return s.policies.List(ctx)
}

func (s *Service) CreatePolicy(ctx context.Context, actor Actor, in CreatePolicyInput) (domain.RetentionPolicy, error) {
	if !canOperate(actor) {
		return domain.RetentionPolicy{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.RetentionPolicy{}, domain.ErrIdempotencyRequired
	}
	in.DataType = strings.TrimSpace(in.DataType)
	if in.DataType == "" || in.RetentionYears < 0 || in.RetentionYears > 99 || in.SoftDeleteGraceDays < 0 || in.SoftDeleteGraceDays > 365 {
		return domain.RetentionPolicy{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{
		"op":                     "create_policy",
		"data_type":              in.DataType,
		"retention_years":        in.RetentionYears,
		"soft_delete_grace_days": in.SoftDeleteGraceDays,
		"selective_rules":        in.SelectiveRules,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RetentionPolicy{}, err
	} else if ok {
		var out domain.RetentionPolicy
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RetentionPolicy{}, err
	}

	row := domain.RetentionPolicy{
		PolicyID:            nextID("pol"),
		DataType:            in.DataType,
		RetentionYears:      in.RetentionYears,
		SoftDeleteGraceDays: in.SoftDeleteGraceDays,
		SelectiveRules:      cloneRules(in.SelectiveRules),
		Status:              domain.PolicyStatusActive,
		CreatedBy:           actor.SubjectID,
		CreatedAt:           s.nowFn(),
	}
	if err := s.policies.Create(ctx, row); err != nil {
		return domain.RetentionPolicy{}, err
	}
	s.appendAudit(ctx, "retention.policy.created", actor.SubjectID, row.PolicyID, map[string]string{"data_type": row.DataType})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) CreatePreview(ctx context.Context, actor Actor, in CreatePreviewInput) (domain.DeletionPreview, error) {
	if !canOperate(actor) {
		return domain.DeletionPreview{}, authorizeError(actor)
	}
	policyID := strings.TrimSpace(in.PolicyID)
	dataType := strings.TrimSpace(in.DataType)
	if policyID == "" && dataType == "" {
		return domain.DeletionPreview{}, domain.ErrInvalidInput
	}
	if policyID != "" {
		policy, err := s.policies.GetByID(ctx, policyID)
		if err != nil {
			return domain.DeletionPreview{}, err
		}
		if dataType == "" {
			dataType = policy.DataType
		}
	}
	if dataType == "" {
		return domain.DeletionPreview{}, domain.ErrInvalidInput
	}

	row := domain.DeletionPreview{
		PreviewID:            nextID("preview"),
		PolicyID:             policyID,
		DataType:             dataType,
		TotalRecordsToDelete: 2500000,
		EstimatedBytes:       512000000,
		WillBeArchivedTo:     "s3://retention-archive/2026/02/" + dataType + "/",
		Status:               domain.PreviewStatusPending,
		RequestedBy:          actor.SubjectID,
		CreatedAt:            s.nowFn(),
	}
	if err := s.previews.Create(ctx, row); err != nil {
		return domain.DeletionPreview{}, err
	}
	s.appendAudit(ctx, "retention.preview.created", actor.SubjectID, row.PreviewID, map[string]string{"data_type": row.DataType})
	return row, nil
}

func (s *Service) ApprovePreview(ctx context.Context, actor Actor, previewID, reason string) (domain.ScheduledDeletion, error) {
	if !canOperate(actor) {
		return domain.ScheduledDeletion{}, authorizeError(actor)
	}
	reason = strings.TrimSpace(reason)
	if strings.TrimSpace(previewID) == "" || reason == "" {
		return domain.ScheduledDeletion{}, domain.ErrInvalidInput
	}
	if existing, ok, err := s.deletions.GetByPreviewID(ctx, strings.TrimSpace(previewID)); err != nil {
		return domain.ScheduledDeletion{}, err
	} else if ok {
		return existing, nil
	}

	preview, err := s.previews.GetByID(ctx, strings.TrimSpace(previewID))
	if err != nil {
		return domain.ScheduledDeletion{}, err
	}
	now := s.nowFn()
	if preview.Status != domain.PreviewStatusApproved {
		preview.Status = domain.PreviewStatusApproved
		preview.ApprovedAt = &now
		if err := s.previews.Update(ctx, preview); err != nil {
			return domain.ScheduledDeletion{}, err
		}
	}

	row := domain.ScheduledDeletion{
		DeletionID:   nextID("del"),
		PreviewID:    preview.PreviewID,
		PolicyID:     preview.PolicyID,
		DataType:     preview.DataType,
		Status:       domain.ScheduledDeletionStatusScheduled,
		RecordsCount: preview.TotalRecordsToDelete,
		Reason:       reason,
		ScheduledAt:  now,
	}
	if err := s.deletions.Create(ctx, row); err != nil {
		return domain.ScheduledDeletion{}, err
	}
	s.appendAudit(ctx, "retention.preview.approved", actor.SubjectID, row.DeletionID, map[string]string{"preview_id": preview.PreviewID})
	return row, nil
}

func (s *Service) CreateLegalHold(ctx context.Context, actor Actor, in CreateLegalHoldInput) (domain.LegalHold, error) {
	if !canOperate(actor) {
		return domain.LegalHold{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.LegalHold{}, domain.ErrIdempotencyRequired
	}
	in.EntityID = strings.TrimSpace(in.EntityID)
	in.DataType = strings.TrimSpace(in.DataType)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.EntityID == "" || in.DataType == "" || in.Reason == "" {
		return domain.LegalHold{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{
		"op":         "create_legal_hold",
		"entity_id":  in.EntityID,
		"data_type":  in.DataType,
		"reason":     in.Reason,
		"expires_at": in.ExpiresAt,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LegalHold{}, err
	} else if ok {
		var out domain.LegalHold
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LegalHold{}, err
	}

	row := domain.LegalHold{
		HoldID:    nextID("hold"),
		EntityID:  in.EntityID,
		DataType:  in.DataType,
		Reason:    in.Reason,
		Status:    domain.LegalHoldStatusActive,
		IssuedBy:  actor.SubjectID,
		CreatedAt: s.nowFn(),
		ExpiresAt: in.ExpiresAt,
	}
	if err := s.holds.Create(ctx, row); err != nil {
		return domain.LegalHold{}, err
	}
	s.appendAudit(ctx, "retention.legal_hold.created", actor.SubjectID, row.HoldID, map[string]string{"entity_id": row.EntityID})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) ListLegalHolds(ctx context.Context, actor Actor, status string) ([]domain.LegalHold, error) {
	if !canView(actor) {
		return nil, authorizeError(actor)
	}
	return s.holds.List(ctx, strings.TrimSpace(status))
}

func (s *Service) CreateRestoration(ctx context.Context, actor Actor, in CreateRestorationInput) (domain.RestorationRequest, error) {
	if !canOperate(actor) {
		return domain.RestorationRequest{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.RestorationRequest{}, domain.ErrIdempotencyRequired
	}
	in.EntityID = strings.TrimSpace(in.EntityID)
	in.DataType = strings.TrimSpace(in.DataType)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.EntityID == "" || in.DataType == "" || in.Reason == "" {
		return domain.RestorationRequest{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{
		"op":               "create_restoration",
		"entity_id":        in.EntityID,
		"data_type":        in.DataType,
		"reason":           in.Reason,
		"archive_location": strings.TrimSpace(in.ArchiveLocation),
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RestorationRequest{}, err
	} else if ok {
		var out domain.RestorationRequest
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.RestorationRequest{}, err
	}

	row := domain.RestorationRequest{
		RestorationID:   nextID("restore"),
		EntityID:        in.EntityID,
		DataType:        in.DataType,
		Reason:          in.Reason,
		ArchiveLocation: strings.TrimSpace(in.ArchiveLocation),
		Status:          domain.RestorationStatusPending,
		RequestedBy:     actor.SubjectID,
		CreatedAt:       s.nowFn(),
	}
	if err := s.restorations.Create(ctx, row); err != nil {
		return domain.RestorationRequest{}, err
	}
	s.appendAudit(ctx, "retention.restoration.created", actor.SubjectID, row.RestorationID, map[string]string{"entity_id": row.EntityID})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) ApproveRestoration(ctx context.Context, actor Actor, restorationID, reason string) (domain.RestorationRequest, error) {
	if !canOperate(actor) {
		return domain.RestorationRequest{}, authorizeError(actor)
	}
	reason = strings.TrimSpace(reason)
	restorationID = strings.TrimSpace(restorationID)
	if restorationID == "" || reason == "" {
		return domain.RestorationRequest{}, domain.ErrInvalidInput
	}
	row, err := s.restorations.GetByID(ctx, restorationID)
	if err != nil {
		return domain.RestorationRequest{}, err
	}
	if row.Status == domain.RestorationStatusApproved {
		return row, nil
	}
	now := s.nowFn()
	row.Status = domain.RestorationStatusApproved
	row.ApprovedAt = &now
	if err := s.restorations.Update(ctx, row); err != nil {
		return domain.RestorationRequest{}, err
	}
	s.appendAudit(ctx, "retention.restoration.approved", actor.SubjectID, row.RestorationID, map[string]string{"reason": reason})
	return row, nil
}

func (s *Service) ComplianceReport(ctx context.Context, actor Actor) (map[string]int, error) {
	if !canView(actor) {
		return nil, authorizeError(actor)
	}
	policies, err := s.policies.List(ctx)
	if err != nil {
		return nil, err
	}
	holds, err := s.holds.List(ctx, domain.LegalHoldStatusActive)
	if err != nil {
		return nil, err
	}
	deletions, err := s.deletions.List(ctx)
	if err != nil {
		return nil, err
	}
	restorations, err := s.restorations.List(ctx)
	if err != nil {
		return nil, err
	}
	totalRecords := 0
	for _, row := range deletions {
		totalRecords += row.RecordsCount
	}
	return map[string]int{
		"policy_count":            len(policies),
		"active_legal_holds":      len(holds),
		"pending_deletions":       len(deletions),
		"total_scheduled_records": totalRecords,
		"restoration_requests":    len(restorations),
	}, nil
}

func canView(actor Actor) bool {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(actor.Role)) {
	case "admin", "compliance", "legal", "support":
		return true
	default:
		return false
	}
}

func canOperate(actor Actor) bool {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(actor.Role)) {
	case "admin", "compliance", "legal":
		return true
	default:
		return false
	}
}

func authorizeError(actor Actor) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ErrUnauthorized
	}
	return domain.ErrForbidden
}

func cloneRules(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func (s *Service) appendAudit(ctx context.Context, eventType, actorID, entityID string, metadata map[string]string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Append(ctx, domain.AuditLog{
		EventID:    nextID("audit"),
		EventType:  eventType,
		ActorID:    actorID,
		EntityID:   entityID,
		OccurredAt: s.nowFn(),
		Metadata:   metadata,
	})
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
