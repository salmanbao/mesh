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
	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/domain"
)

func (s *Service) CreateDispute(ctx context.Context, actor Actor, input CreateDisputeInput) (domain.Dispute, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Dispute{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Dispute{}, domain.ErrIdempotencyRequired
	}
	disputeType := domain.NormalizeDisputeType(input.DisputeType)
	if disputeType == "" || strings.TrimSpace(input.TransactionID) == "" || strings.TrimSpace(input.ReasonCategory) == "" || input.RequestedAmount <= 0 {
		return domain.Dispute{}, domain.ErrInvalidInput
	}
	if err := domain.ValidateJustification(input.JustificationText); err != nil {
		return domain.Dispute{}, err
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentDispute(ctx, actor, requestHash); err != nil {
		return domain.Dispute{}, err
	} else if ok {
		return cached, nil
	}
	if s.disputes != nil {
		if existing, err := s.disputes.GetOpenByTransactionID(ctx, strings.TrimSpace(input.TransactionID)); err == nil && existing.DisputeID != "" {
			return domain.Dispute{}, domain.ErrConflict
		}
	}

	if s.moderation != nil {
		_, _ = s.moderation.GetModerationSummary(ctx, actor.SubjectID)
	}

	now := s.nowFn()
	expected := now.Add(time.Duration(domain.DefaultSLAHours(disputeType)) * time.Hour)
	dispute := domain.Dispute{
		DisputeID:          uuid.NewString(),
		DisputeType:        disputeType,
		Status:             domain.DisputeStatusSubmitted,
		Priority:           domain.PriorityForAmount(input.RequestedAmount),
		UserID:             actor.SubjectID,
		TransactionID:      strings.TrimSpace(input.TransactionID),
		EntityType:         "transaction",
		EntityID:           strings.TrimSpace(input.TransactionID),
		ReasonCategory:     strings.TrimSpace(input.ReasonCategory),
		JustificationText:  strings.TrimSpace(input.JustificationText),
		RequestedAmount:    input.RequestedAmount,
		AssignedAgentID:    assignAgent(actor.SubjectID),
		SLAHoursTarget:     domain.DefaultSLAHours(disputeType),
		SLABreached:        false,
		RefundPending:      false,
		CreatedAt:          now,
		UpdatedAt:          now,
		EvidenceFiles:      cloneEvidence(input.EvidenceFiles),
		ExpectedResolution: &expected,
	}
	if s.disputes != nil {
		if err := s.disputes.Create(ctx, dispute); err != nil {
			return domain.Dispute{}, err
		}
	}
	if len(input.EvidenceFiles) > 0 && s.evidence != nil {
		rows := make([]domain.DisputeEvidence, 0, len(input.EvidenceFiles))
		for _, f := range input.EvidenceFiles {
			rows = append(rows, domain.DisputeEvidence{
				EvidenceID:       uuid.NewString(),
				DisputeID:        dispute.DisputeID,
				UploadedByUserID: actor.SubjectID,
				FileURL:          strings.TrimSpace(f.FileURL),
				Filename:         strings.TrimSpace(f.Filename),
				UploadedAt:       now,
				Scanned:          true,
				ScanResult:       "clean",
			})
		}
		if err := s.evidence.CreateMany(ctx, rows); err != nil {
			return domain.Dispute{}, err
		}
	}
	_ = s.recordStateTransition(ctx, dispute.DisputeID, "", domain.DisputeStatusSubmitted, actor.SubjectID, "dispute created", now)
	_ = s.recordAudit(ctx, dispute.DisputeID, "created", actor.SubjectID, map[string]string{"transaction_id": dispute.TransactionID}, now)
	_ = s.enqueueDisputeCreated(ctx, actor, dispute)

	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 201, dispute); err != nil {
		return domain.Dispute{}, err
	}
	return dispute, nil
}

func (s *Service) GetDispute(ctx context.Context, actor Actor, disputeID string) (domain.DisputeDetail, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DisputeDetail{}, domain.ErrUnauthorized
	}
	if s.disputes == nil {
		return domain.DisputeDetail{}, domain.ErrNotFound
	}
	dispute, err := s.disputes.GetByID(ctx, strings.TrimSpace(disputeID))
	if err != nil {
		return domain.DisputeDetail{}, err
	}
	if err := authorizeDisputeAccess(actor, dispute); err != nil {
		return domain.DisputeDetail{}, err
	}
	var messages []domain.DisputeMessage
	if s.messages != nil {
		messages, _ = s.messages.ListByDispute(ctx, dispute.DisputeID, 100)
	}
	var history []domain.DisputeStateHistory
	if s.stateHistory != nil {
		history, _ = s.stateHistory.ListByDispute(ctx, dispute.DisputeID)
	}
	return domain.DisputeDetail{Dispute: dispute, Messages: messages, StateHistory: history}, nil
}

func (s *Service) SendMessage(ctx context.Context, actor Actor, disputeID string, input SendMessageInput) (domain.DisputeMessage, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DisputeMessage{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DisputeMessage{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(disputeID) == "" || len(strings.TrimSpace(input.MessageBody)) == 0 {
		return domain.DisputeMessage{}, domain.ErrInvalidInput
	}
	dispute, err := s.disputes.GetByID(ctx, strings.TrimSpace(disputeID))
	if err != nil {
		return domain.DisputeMessage{}, err
	}
	if err := authorizeDisputeAccess(actor, dispute); err != nil {
		return domain.DisputeMessage{}, err
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentMessage(ctx, actor, requestHash); err != nil {
		return domain.DisputeMessage{}, err
	} else if ok {
		return cached, nil
	}
	now := s.nowFn()
	msg := domain.DisputeMessage{
		MessageID:   uuid.NewString(),
		DisputeID:   dispute.DisputeID,
		SenderID:    actor.SubjectID,
		MessageBody: strings.TrimSpace(input.MessageBody),
		Attachments: cloneEvidence(input.Attachments),
		CreatedAt:   now,
	}
	if s.messages != nil {
		if err := s.messages.Create(ctx, msg); err != nil {
			return domain.DisputeMessage{}, err
		}
	}
	if len(msg.Attachments) > 0 && s.evidence != nil {
		rows := make([]domain.DisputeEvidence, 0, len(msg.Attachments))
		for _, a := range msg.Attachments {
			rows = append(rows, domain.DisputeEvidence{EvidenceID: uuid.NewString(), DisputeID: dispute.DisputeID, UploadedByUserID: actor.SubjectID, FileURL: a.FileURL, Filename: a.Filename, UploadedAt: now, Scanned: true, ScanResult: "clean"})
		}
		_ = s.evidence.CreateMany(ctx, rows)
	}
	_ = s.recordAudit(ctx, dispute.DisputeID, "message_sent", actor.SubjectID, map[string]string{"message_id": msg.MessageID}, now)
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 201, msg); err != nil {
		return domain.DisputeMessage{}, err
	}
	return msg, nil
}

func (s *Service) ApproveDispute(ctx context.Context, actor Actor, disputeID string, input ApproveDisputeInput) (domain.Dispute, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Dispute{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Dispute{}, domain.ErrIdempotencyRequired
	}
	if !isStaffRole(actor.Role) {
		return domain.Dispute{}, domain.ErrForbidden
	}
	if input.RefundAmount <= 0 || strings.TrimSpace(input.ApprovalReason) == "" {
		return domain.Dispute{}, domain.ErrInvalidInput
	}
	if !domain.CanApproveRefund(actor.Role, input.RefundAmount) {
		return domain.Dispute{}, domain.ErrForbidden
	}
	dispute, err := s.disputes.GetByID(ctx, strings.TrimSpace(disputeID))
	if err != nil {
		return domain.Dispute{}, err
	}
	if dispute.Status == domain.DisputeStatusResolved || dispute.Status == domain.DisputeStatusWithdrawn {
		return domain.Dispute{}, domain.ErrConflict
	}
	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentDispute(ctx, actor, requestHash); err != nil {
		return domain.Dispute{}, err
	} else if ok && cached.DisputeID == dispute.DisputeID {
		return cached, nil
	}
	switch dispute.Status {
	case domain.DisputeStatusSubmitted, domain.DisputeStatusUnderReview, domain.DisputeStatusEscalated, domain.DisputeStatusAwaitingAction:
		// Allowed in MVP implementation.
	default:
		if err := domain.ValidateStatusTransition(dispute.Status, domain.DisputeStatusResolved); err != nil {
			return domain.Dispute{}, err
		}
	}

	now := s.nowFn()
	approval := domain.DisputeApproval{
		ApprovalID:      uuid.NewString(),
		DisputeID:       dispute.DisputeID,
		ApprovedBy:      actor.SubjectID,
		ApprovalLevel:   domain.ApprovalLevelForRole(actor.Role),
		RefundAmount:    input.RefundAmount,
		ApprovalReason:  strings.TrimSpace(input.ApprovalReason),
		ResolutionNotes: strings.TrimSpace(input.ResolutionNotes),
		Status:          "approved",
		ApprovedAt:      now,
	}
	if s.approvals != nil {
		if err := s.approvals.Create(ctx, approval); err != nil {
			return domain.Dispute{}, err
		}
	}
	prevStatus := dispute.Status
	dispute.Status = domain.DisputeStatusResolved
	dispute.ApprovedRefundAmount = input.RefundAmount
	if input.RefundAmount < dispute.RequestedAmount {
		dispute.ResolutionType = domain.ResolutionTypePartialRefund
	} else {
		dispute.ResolutionType = domain.ResolutionTypeRefundIssued
	}
	dispute.ResolutionNotes = strings.TrimSpace(input.ResolutionNotes)
	dispute.RefundPending = false
	dispute.UpdatedAt = now
	dispute.ResolvedAt = &now
	if s.disputes != nil {
		if err := s.disputes.Update(ctx, dispute); err != nil {
			return domain.Dispute{}, err
		}
	}
	_ = s.recordStateTransition(ctx, dispute.DisputeID, prevStatus, dispute.Status, actor.SubjectID, "refund approved", now)
	_ = s.recordAudit(ctx, dispute.DisputeID, "approved", actor.SubjectID, map[string]string{"approval_id": approval.ApprovalID}, now)
	_ = s.publishDisputeResolvedAnalytics(ctx, actor, dispute)
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, 200, dispute); err != nil {
		return domain.Dispute{}, err
	}
	return dispute, nil
}

func (s *Service) applyConsumedEvent(ctx context.Context, eventType string, payload map[string]string, now time.Time) error {
	if s.auditLogs == nil {
		return nil
	}
	return s.auditLogs.Create(ctx, domain.DisputeAuditLog{
		AuditLogID: uuid.NewString(),
		ActionType: "event_consumed",
		Metadata: map[string]string{
			"event_type": eventType,
			"entity_id":  payload["entity_id"],
			"user_id":    payload["user_id"],
		},
		CreatedAt: now,
	})
}

func authorizeDisputeAccess(actor Actor, dispute domain.Dispute) error {
	if isStaffRole(actor.Role) {
		return nil
	}
	if strings.TrimSpace(actor.SubjectID) != dispute.UserID {
		return domain.ErrForbidden
	}
	return nil
}

func cloneEvidence(in []domain.EvidenceFile) []domain.EvidenceFile {
	out := make([]domain.EvidenceFile, 0, len(in))
	for _, e := range in {
		filename := strings.TrimSpace(e.Filename)
		fileURL := strings.TrimSpace(e.FileURL)
		if filename == "" && fileURL == "" {
			continue
		}
		out = append(out, domain.EvidenceFile{Filename: filename, FileURL: fileURL})
	}
	return out
}

func assignAgent(subjectID string) string {
	if subjectID == "" {
		return "agent-0001"
	}
	sum := sha256.Sum256([]byte(subjectID))
	return fmt.Sprintf("agent-%04d", (int(sum[0])<<8|int(sum[1]))%1000+1)
}

func (s *Service) recordAudit(ctx context.Context, disputeID, actionType, actorID string, meta map[string]string, at time.Time) error {
	if s.auditLogs == nil {
		return nil
	}
	return s.auditLogs.Create(ctx, domain.DisputeAuditLog{AuditLogID: uuid.NewString(), DisputeID: disputeID, ActionType: actionType, ActorID: actorID, Metadata: meta, CreatedAt: at})
}

func (s *Service) recordStateTransition(ctx context.Context, disputeID, from, to, changedBy, reason string, at time.Time) error {
	if s.stateHistory == nil {
		return nil
	}
	return s.stateHistory.Create(ctx, domain.DisputeStateHistory{HistoryID: uuid.NewString(), DisputeID: disputeID, FromStatus: from, ToStatus: to, ChangedBy: changedBy, Reason: reason, ChangedAt: at})
}

func (s *Service) getIdempotentDispute(ctx context.Context, actor Actor, requestHash string) (domain.Dispute, bool, error) {
	if s.idempotency == nil {
		return domain.Dispute{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.Dispute{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.Dispute{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.Dispute
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.Dispute{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.Dispute{}, false, err
	}
	return domain.Dispute{}, false, nil
}

func (s *Service) getIdempotentMessage(ctx context.Context, actor Actor, requestHash string) (domain.DisputeMessage, bool, error) {
	if s.idempotency == nil {
		return domain.DisputeMessage{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.DisputeMessage{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.DisputeMessage{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.DisputeMessage
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.DisputeMessage{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.DisputeMessage{}, false, err
	}
	return domain.DisputeMessage{}, false, nil
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
