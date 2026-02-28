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

	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) RegisterDeveloper(ctx context.Context, actor Actor, in RegisterDeveloperInput) (domain.Developer, domain.DeveloperSession, error) {
	if !canRegister(actor) {
		return domain.Developer{}, domain.DeveloperSession{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Developer{}, domain.DeveloperSession{}, domain.ErrIdempotencyRequired
	}
	in.Email = strings.TrimSpace(in.Email)
	in.AppName = strings.TrimSpace(in.AppName)
	if in.Email == "" || in.AppName == "" {
		return domain.Developer{}, domain.DeveloperSession{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "register_developer", "email": in.Email, "app_name": in.AppName})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Developer{}, domain.DeveloperSession{}, err
	} else if ok {
		var out struct {
			Developer domain.Developer        `json:"developer"`
			Session   domain.DeveloperSession `json:"session"`
		}
		if json.Unmarshal(raw, &out) == nil {
			return out.Developer, out.Session, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Developer{}, domain.DeveloperSession{}, err
	}
	now := s.nowFn()
	developer := domain.Developer{
		DeveloperID: nextID("dev"),
		Email:       in.Email,
		AppName:     in.AppName,
		Tier:        "free",
		Status:      domain.DeveloperStatusActive,
		CreatedAt:   now,
	}
	session := domain.DeveloperSession{
		SessionID:    nextID("sess"),
		DeveloperID:  developer.DeveloperID,
		SessionToken: "tok_" + developer.DeveloperID,
		Status:       domain.SessionStatusActive,
		ExpiresAt:    now.Add(1 * time.Hour),
		CreatedAt:    now,
	}
	if err := s.developers.Create(ctx, developer); err != nil {
		return domain.Developer{}, domain.DeveloperSession{}, err
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return domain.Developer{}, domain.DeveloperSession{}, err
	}
	if s.usage != nil {
		_ = s.usage.CreateOrUpdate(ctx, domain.DeveloperUsage{
			UsageID:      nextID("usage"),
			DeveloperID:  developer.DeveloperID,
			CurrentUsage: 0,
			RateLimit:    100,
			PeriodStart:  now.Truncate(time.Hour),
			PeriodEnd:    now.Truncate(time.Hour).Add(time.Hour),
		})
	}
	s.appendAudit(ctx, developer.DeveloperID, "developer.account_created", developer.DeveloperID, nil)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, struct {
		Developer domain.Developer        `json:"developer"`
		Session   domain.DeveloperSession `json:"session"`
	}{Developer: developer, Session: session})
	return developer, session, nil
}

func (s *Service) CreateAPIKey(ctx context.Context, actor Actor, in CreateAPIKeyInput) (domain.APIKey, error) {
	if !canOperate(actor) {
		return domain.APIKey{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.APIKey{}, domain.ErrIdempotencyRequired
	}
	developerID, err := s.resolveDeveloperID(actor, in.DeveloperID)
	if err != nil {
		return domain.APIKey{}, err
	}
	in.Label = strings.TrimSpace(in.Label)
	if in.Label == "" {
		return domain.APIKey{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "create_api_key", "developer_id": developerID, "label": in.Label})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.APIKey{}, err
	} else if ok {
		var out domain.APIKey
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.APIKey{}, err
	}
	if _, err := s.developers.GetByID(ctx, developerID); err != nil {
		return domain.APIKey{}, err
	}
	row := domain.APIKey{
		KeyID:       nextID("key"),
		DeveloperID: developerID,
		Label:       in.Label,
		MaskedKey:   "vk_live_" + developerID + "_****",
		Status:      domain.APIKeyStatusActive,
		CreatedAt:   s.nowFn(),
	}
	if err := s.apiKeys.Create(ctx, row); err != nil {
		return domain.APIKey{}, err
	}
	s.appendAudit(ctx, developerID, "developer.key_generated", row.KeyID, nil)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) RotateAPIKey(ctx context.Context, actor Actor, keyID string) (domain.APIKeyRotation, domain.APIKey, domain.APIKey, error) {
	if !canOperate(actor) {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, domain.ErrIdempotencyRequired
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "rotate_api_key", "key_id": keyID})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, err
	} else if ok {
		var out struct {
			Rotation domain.APIKeyRotation `json:"rotation"`
			OldKey   domain.APIKey         `json:"old_key"`
			NewKey   domain.APIKey         `json:"new_key"`
		}
		if json.Unmarshal(raw, &out) == nil {
			return out.Rotation, out.OldKey, out.NewKey, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, err
	}
	oldKey, err := s.apiKeys.GetByID(ctx, keyID)
	if err != nil {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, err
	}
	if !canAccessDeveloper(actor, oldKey.DeveloperID) {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, authorizeError(actor)
	}
	oldKey.Status = domain.APIKeyStatusDeprecated
	if err := s.apiKeys.Update(ctx, oldKey); err != nil {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, err
	}
	newKey := domain.APIKey{
		KeyID:       nextID("key"),
		DeveloperID: oldKey.DeveloperID,
		Label:       oldKey.Label + " (rotated)",
		MaskedKey:   "vk_live_" + oldKey.DeveloperID + "_****",
		Status:      domain.APIKeyStatusActive,
		CreatedAt:   s.nowFn(),
	}
	if err := s.apiKeys.Create(ctx, newKey); err != nil {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, err
	}
	rotation := domain.APIKeyRotation{
		RotationID:  nextID("rot"),
		OldKeyID:    oldKey.KeyID,
		NewKeyID:    newKey.KeyID,
		DeveloperID: oldKey.DeveloperID,
		CreatedAt:   s.nowFn(),
	}
	if err := s.rotations.Create(ctx, rotation); err != nil {
		return domain.APIKeyRotation{}, domain.APIKey{}, domain.APIKey{}, err
	}
	s.appendAudit(ctx, oldKey.DeveloperID, "developer.key_rotated", rotation.RotationID, nil)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, struct {
		Rotation domain.APIKeyRotation `json:"rotation"`
		OldKey   domain.APIKey         `json:"old_key"`
		NewKey   domain.APIKey         `json:"new_key"`
	}{Rotation: rotation, OldKey: oldKey, NewKey: newKey})
	return rotation, oldKey, newKey, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, actor Actor, keyID string) (domain.APIKey, error) {
	if !canOperate(actor) {
		return domain.APIKey{}, authorizeError(actor)
	}
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return domain.APIKey{}, domain.ErrInvalidInput
	}
	row, err := s.apiKeys.GetByID(ctx, keyID)
	if err != nil {
		return domain.APIKey{}, err
	}
	if !canAccessDeveloper(actor, row.DeveloperID) {
		return domain.APIKey{}, authorizeError(actor)
	}
	if row.Status == domain.APIKeyStatusRevoked {
		return row, nil
	}
	now := s.nowFn()
	row.Status = domain.APIKeyStatusRevoked
	row.RevokedAt = &now
	if err := s.apiKeys.Update(ctx, row); err != nil {
		return domain.APIKey{}, err
	}
	s.appendAudit(ctx, row.DeveloperID, "developer.key_revoked", row.KeyID, nil)
	return row, nil
}

func (s *Service) CreateWebhook(ctx context.Context, actor Actor, in CreateWebhookInput) (domain.Webhook, error) {
	if !canOperate(actor) {
		return domain.Webhook{}, authorizeError(actor)
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Webhook{}, domain.ErrIdempotencyRequired
	}
	developerID, err := s.resolveDeveloperID(actor, in.DeveloperID)
	if err != nil {
		return domain.Webhook{}, err
	}
	in.URL = strings.TrimSpace(in.URL)
	in.EventType = strings.TrimSpace(in.EventType)
	if in.URL == "" || in.EventType == "" {
		return domain.Webhook{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]string{"op": "create_webhook", "developer_id": developerID, "url": in.URL, "event_type": in.EventType})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	} else if ok {
		var out domain.Webhook
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	}
	if _, err := s.developers.GetByID(ctx, developerID); err != nil {
		return domain.Webhook{}, err
	}
	row := domain.Webhook{
		WebhookID:   nextID("wh"),
		DeveloperID: developerID,
		URL:         in.URL,
		EventType:   in.EventType,
		Status:      domain.WebhookStatusActive,
		CreatedAt:   s.nowFn(),
	}
	if err := s.webhooks.Create(ctx, row); err != nil {
		return domain.Webhook{}, err
	}
	s.appendAudit(ctx, developerID, "developer.webhook_created", row.WebhookID, nil)
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) TestWebhook(ctx context.Context, actor Actor, webhookID string) (domain.WebhookDelivery, error) {
	if !canOperate(actor) {
		return domain.WebhookDelivery{}, authorizeError(actor)
	}
	webhookID = strings.TrimSpace(webhookID)
	if webhookID == "" {
		return domain.WebhookDelivery{}, domain.ErrInvalidInput
	}
	wh, err := s.webhooks.GetByID(ctx, webhookID)
	if err != nil {
		return domain.WebhookDelivery{}, err
	}
	if !canAccessDeveloper(actor, wh.DeveloperID) {
		return domain.WebhookDelivery{}, authorizeError(actor)
	}
	row := domain.WebhookDelivery{
		DeliveryID: nextID("delivery"),
		WebhookID:  wh.WebhookID,
		Status:     domain.DeliveryStatusSuccess,
		TestEvent:  true,
		CreatedAt:  s.nowFn(),
	}
	if err := s.deliveries.Create(ctx, row); err != nil {
		return domain.WebhookDelivery{}, err
	}
	s.appendAudit(ctx, wh.DeveloperID, "developer.webhook_tested", row.DeliveryID, nil)
	return row, nil
}

func (s *Service) resolveDeveloperID(actor Actor, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if requested == "" {
		return "", domain.ErrUnauthorized
	}
	if !canAccessDeveloper(actor, requested) {
		return "", authorizeError(actor)
	}
	return requested, nil
}

func canRegister(actor Actor) bool {
	return strings.TrimSpace(actor.SubjectID) != ""
}

func canOperate(actor Actor) bool {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	return role == "admin" || role == "developer" || role == "user"
}

func canAccessDeveloper(actor Actor, developerID string) bool {
	if strings.TrimSpace(actor.SubjectID) == "" || strings.TrimSpace(developerID) == "" {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	return actor.SubjectID == developerID || role == "admin" || role == "support"
}

func authorizeError(actor Actor) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ErrUnauthorized
	}
	return domain.ErrForbidden
}

func (s *Service) appendAudit(ctx context.Context, developerID, actionType, entityID string, metadata map[string]string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Append(ctx, domain.AuditLog{
		AuditID:     nextID("audit"),
		DeveloperID: developerID,
		ActionType:  actionType,
		EntityID:    entityID,
		OccurredAt:  s.nowFn(),
		Metadata:    metadata,
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
