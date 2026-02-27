package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M03-notification-service/internal/domain"
)

func (s *Service) ListNotifications(ctx context.Context, actor Actor, input ListNotificationsInput) ([]domain.Notification, int, error) {
	userID, err := s.resolveUser(actor, input.UserID)
	if err != nil {
		return nil, 0, err
	}
	filter := domain.NotificationFilter{
		Type:     strings.TrimSpace(input.Type),
		Status:   strings.ToLower(strings.TrimSpace(input.Status)),
		Page:     input.Page,
		PageSize: input.PageSize,
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	return s.notifications.ListByUserID(ctx, userID, filter)
}

func (s *Service) UnreadCount(ctx context.Context, actor Actor, userID string) (int, string, error) {
	resolved, err := s.resolveUser(actor, userID)
	if err != nil {
		return 0, "", err
	}
	count, err := s.notifications.CountUnread(ctx, resolved)
	return count, resolved, err
}

func (s *Service) MarkRead(ctx context.Context, actor Actor, notificationID string) (domain.Notification, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Notification{}, domain.ErrUnauthorized
	}
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return domain.Notification{}, domain.ErrInvalidInput
	}
	row, err := s.notifications.GetByID(ctx, notificationID)
	if err != nil {
		return domain.Notification{}, err
	}
	if !canActForUser(actor, row.UserID) {
		return domain.Notification{}, domain.ErrForbidden
	}
	row.MarkRead(s.nowFn())
	if err := s.notifications.Update(ctx, row); err != nil {
		return domain.Notification{}, err
	}
	return row, nil
}

func (s *Service) Archive(ctx context.Context, actor Actor, notificationID string) (domain.Notification, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Notification{}, domain.ErrUnauthorized
	}
	notificationID = strings.TrimSpace(notificationID)
	if notificationID == "" {
		return domain.Notification{}, domain.ErrInvalidInput
	}
	row, err := s.notifications.GetByID(ctx, notificationID)
	if err != nil {
		return domain.Notification{}, err
	}
	if !canActForUser(actor, row.UserID) {
		return domain.Notification{}, domain.ErrForbidden
	}
	row.Archive(s.nowFn())
	if err := s.notifications.Update(ctx, row); err != nil {
		return domain.Notification{}, err
	}
	return row, nil
}

func (s *Service) BulkAction(ctx context.Context, actor Actor, input BulkActionInput) (int, error) {
	userID, err := s.resolveUser(actor, input.UserID)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return 0, domain.ErrIdempotencyRequired
	}
	if len(input.NotificationIDs) == 0 {
		return 0, domain.ErrInvalidInput
	}
	if len(input.NotificationIDs) > 100 {
		return 0, domain.ErrPayloadTooLarge
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action != "mark_read" && action != "archive" {
		return 0, domain.ErrInvalidInput
	}
	cleanIDs := make([]string, 0, len(input.NotificationIDs))
	for _, id := range input.NotificationIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return 0, domain.ErrInvalidInput
		}
		cleanIDs = append(cleanIDs, id)
	}
	requestHash := hashJSON(map[string]any{"op": "bulk_action", "user_id": userID, "action": action, "ids": cleanIDs})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return 0, err
	} else if ok {
		var payload struct {
			Updated int `json:"updated"`
		}
		if json.Unmarshal(rec, &payload) == nil {
			return payload.Updated, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return 0, err
	}
	updated := 0
	for _, id := range cleanIDs {
		row, err := s.notifications.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if row.UserID != userID {
			continue
		}
		switch action {
		case "mark_read":
			row.MarkRead(s.nowFn())
		case "archive":
			row.Archive(s.nowFn())
		}
		if err := s.notifications.Update(ctx, row); err == nil {
			updated++
		}
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, map[string]int{"updated": updated})
	return updated, nil
}

func (s *Service) GetPreferences(ctx context.Context, actor Actor, userID string) (domain.Preferences, error) {
	resolved, err := s.resolveUser(actor, userID)
	if err != nil {
		return domain.Preferences{}, err
	}
	row, err := s.preferences.GetByUserID(ctx, resolved)
	if err == nil {
		return row, nil
	}
	return domain.DefaultPreferences(resolved, s.nowFn()), nil
}

func (s *Service) UpdatePreferences(ctx context.Context, actor Actor, input UpdatePreferencesInput) (domain.Preferences, error) {
	userID, err := s.resolveUser(actor, input.UserID)
	if err != nil {
		return domain.Preferences{}, err
	}
	row, err := s.preferences.GetByUserID(ctx, userID)
	if err != nil {
		row = domain.DefaultPreferences(userID, s.nowFn())
	}
	if input.EmailEnabled != nil {
		row.EmailEnabled = *input.EmailEnabled
	}
	if input.PushEnabled != nil {
		row.PushEnabled = *input.PushEnabled
	}
	if input.SMSEnabled != nil {
		row.SMSEnabled = *input.SMSEnabled
	}
	if input.InAppEnabled != nil {
		row.InAppEnabled = *input.InAppEnabled
	}
	if input.QuietHoursEnabled != nil {
		row.QuietHoursEnabled = *input.QuietHoursEnabled
	}
	if strings.TrimSpace(input.QuietHoursStart) != "" {
		row.QuietHoursStart = strings.TrimSpace(input.QuietHoursStart)
	}
	if strings.TrimSpace(input.QuietHoursEnd) != "" {
		row.QuietHoursEnd = strings.TrimSpace(input.QuietHoursEnd)
	}
	if input.MutedTypes != nil {
		row.MutedTypes = sanitizeStringSlice(input.MutedTypes)
	}
	row.UpdatedAt = s.nowFn()
	if err := s.preferences.Upsert(ctx, row); err != nil {
		return domain.Preferences{}, err
	}
	return row, nil
}

func (s *Service) CancelScheduled(ctx context.Context, actor Actor, scheduledID string) (bool, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false, domain.ErrUnauthorized
	}
	if strings.ToLower(strings.TrimSpace(actor.Role)) != "admin" {
		return false, domain.ErrForbidden
	}
	scheduledID = strings.TrimSpace(scheduledID)
	if scheduledID == "" {
		return false, domain.ErrInvalidInput
	}
	return s.scheduled.Delete(ctx, scheduledID)
}

func (s *Service) FlushOutbox(context.Context) error { return nil }

func canActForUser(actor Actor, userID string) bool {
	actorID := strings.TrimSpace(actor.SubjectID)
	userID = strings.TrimSpace(userID)
	if actorID == "" || userID == "" {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	return actorID == userID || role == "admin" || role == "support"
}

func (s *Service) resolveUser(actor Actor, requestedUserID string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requestedUserID = strings.TrimSpace(requestedUserID)
	if requestedUserID == "" {
		requestedUserID = strings.TrimSpace(actor.SubjectID)
	}
	if !canActForUser(actor, requestedUserID) {
		return "", domain.ErrForbidden
	}
	return requestedUserID, nil
}

func sanitizeStringSlice(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
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

func newNotificationID() string { return "notif-" + uuid.NewString() }
