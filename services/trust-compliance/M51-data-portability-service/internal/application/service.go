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

	"github.com/viralforge/mesh/services/trust-compliance/M51-data-portability-service/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) CreateExport(ctx context.Context, actor Actor, in CreateExportInput) (domain.ExportRequest, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.ExportRequest{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ExportRequest{}, domain.ErrIdempotencyRequired
	}
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		return domain.ExportRequest{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]string{"op": "create_export", "user_id": userID, "format": format})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportRequest{}, err
	} else if ok {
		var out domain.ExportRequest
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportRequest{}, err
	}

	now := s.nowFn()
	row := domain.ExportRequest{
		RequestID:   nextID("exp"),
		UserID:      userID,
		RequestType: domain.ExportRequestTypeExport,
		Format:      format,
		Status:      domain.ExportStatusPending,
		RequestedAt: now,
	}
	if err := s.exports.Create(ctx, row); err != nil {
		return domain.ExportRequest{}, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:    nextID("audit"),
			EventType:  "export.requested",
			RequestID:  row.RequestID,
			UserID:     row.UserID,
			ActorID:    actor.SubjectID,
			OccurredAt: now,
		})
	}

	doneAt := s.nowFn()
	row.Status = domain.ExportStatusCompleted
	row.CompletedAt = &doneAt
	row.DownloadURL = "https://downloads.example.com/v1/exports/" + row.RequestID
	if err := s.exports.Update(ctx, row); err != nil {
		return domain.ExportRequest{}, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:    nextID("audit"),
			EventType:  domain.EventExportCompleted,
			RequestID:  row.RequestID,
			UserID:     row.UserID,
			ActorID:    actor.SubjectID,
			OccurredAt: doneAt,
		})
	}

	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) CreateEraseRequest(ctx context.Context, actor Actor, in EraseInput) (domain.ExportRequest, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.ExportRequest{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ExportRequest{}, domain.ErrIdempotencyRequired
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return domain.ExportRequest{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]string{"op": "erase_export", "user_id": userID, "reason": reason})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportRequest{}, err
	} else if ok {
		var out domain.ExportRequest
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.ExportRequest{}, err
	}

	now := s.nowFn()
	row := domain.ExportRequest{
		RequestID:   nextID("era"),
		UserID:      userID,
		RequestType: domain.ExportRequestTypeErase,
		Status:      domain.ExportStatusCompleted,
		Reason:      reason,
		RequestedAt: now,
		CompletedAt: &now,
	}
	if err := s.exports.Create(ctx, row); err != nil {
		return domain.ExportRequest{}, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:    nextID("audit"),
			EventType:  domain.EventExportCompleted,
			RequestID:  row.RequestID,
			UserID:     row.UserID,
			ActorID:    actor.SubjectID,
			OccurredAt: now,
			Metadata:   map[string]string{"request_type": domain.ExportRequestTypeErase},
		})
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) GetExport(ctx context.Context, actor Actor, requestID string) (domain.ExportRequest, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportRequest{}, domain.ErrUnauthorized
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return domain.ExportRequest{}, domain.ErrInvalidInput
	}
	row, err := s.exports.GetByID(ctx, requestID)
	if err != nil {
		return domain.ExportRequest{}, err
	}
	if !canActForUser(actor, row.UserID) {
		return domain.ExportRequest{}, domain.ErrForbidden
	}
	return row, nil
}

func (s *Service) ListExports(ctx context.Context, actor Actor, userID string, limit int) ([]domain.ExportRequest, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if limit <= 0 {
		limit = 50
	}
	userID = strings.TrimSpace(userID)
	if userID == "" || !isPrivileged(actor) {
		userID = strings.TrimSpace(actor.SubjectID)
	}
	if !canActForUser(actor, userID) {
		return nil, domain.ErrForbidden
	}
	return s.exports.ListByUserID(ctx, userID, limit)
}

func canActForUser(actor Actor, userID string) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	actorID := strings.TrimSpace(actor.SubjectID)
	userID = strings.TrimSpace(userID)
	return actorID != "" && userID != "" && (actorID == userID || role == "admin" || role == "support" || role == "legal")
}

func isPrivileged(actor Actor) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	return role == "admin" || role == "support" || role == "legal"
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
