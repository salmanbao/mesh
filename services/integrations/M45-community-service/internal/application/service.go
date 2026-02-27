package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/integrations/M45-community-service/internal/domain"
)

func (s *Service) ConnectIntegration(ctx context.Context, actor Actor, in ConnectIntegrationInput) (domain.CommunityIntegration, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CommunityIntegration{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CommunityIntegration{}, domain.ErrIdempotencyRequired
	}
	platform := strings.ToLower(strings.TrimSpace(in.Platform))
	communityName := strings.TrimSpace(in.CommunityName)
	cfg := sanitizeMap(in.Config)
	if !domain.IsValidPlatform(platform) || communityName == "" || len(cfg) == 0 {
		return domain.CommunityIntegration{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(map[string]any{"op": "connect_integration", "creator_id": actor.SubjectID, "platform": platform, "community_name": communityName, "config": cfg})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CommunityIntegration{}, err
	} else if ok {
		var out domain.CommunityIntegration
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CommunityIntegration{}, err
	}
	now := s.nowFn()
	row := domain.CommunityIntegration{IntegrationID: "int_" + uuid.NewString(), CreatorID: actor.SubjectID, Platform: platform, CommunityName: communityName, Config: cfg, Status: domain.IntegrationStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.integrations.Create(ctx, row); err != nil {
		return domain.CommunityIntegration{}, err
	}
	if s.healthChecks != nil {
		_ = s.healthChecks.Append(ctx, domain.CommunityHealthCheck{HealthCheckID: "hc_" + uuid.NewString(), IntegrationID: row.IntegrationID, CheckedAt: now, Status: domain.HealthStatusHealthy, LatencyMS: 120, HTTPStatusCode: 200})
	}
	_ = s.appendAudit(ctx, actor, "community.integration.connected", row.CreatorID, row.IntegrationID, "", "", "success", "", map[string]string{"platform": row.Platform, "community_name": row.CommunityName})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) GetIntegration(ctx context.Context, actor Actor, integrationID string) (domain.CommunityIntegration, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CommunityIntegration{}, domain.ErrUnauthorized
	}
	integrationID = strings.TrimSpace(integrationID)
	if integrationID == "" {
		return domain.CommunityIntegration{}, domain.ErrInvalidInput
	}
	row, err := s.integrations.GetByID(ctx, integrationID)
	if err != nil {
		return domain.CommunityIntegration{}, err
	}
	if !canReadCreatorResource(actor, row.CreatorID) {
		return domain.CommunityIntegration{}, domain.ErrForbidden
	}
	return row, nil
}

func (s *Service) CreateManualGrant(ctx context.Context, actor Actor, in ManualGrantInput) (domain.CommunityGrant, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CommunityGrant{}, domain.ErrUnauthorized
	}
	if !isAdmin(actor) {
		return domain.CommunityGrant{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CommunityGrant{}, domain.ErrIdempotencyRequired
	}
	in.UserID = strings.TrimSpace(in.UserID)
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.IntegrationID = strings.TrimSpace(in.IntegrationID)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Tier = strings.ToLower(strings.TrimSpace(in.Tier))
	if in.Tier == "" {
		in.Tier = "basic"
	}
	if in.UserID == "" || in.ProductID == "" || in.IntegrationID == "" || in.Reason == "" {
		return domain.CommunityGrant{}, domain.ErrInvalidInput
	}
	integration, err := s.integrations.GetByID(ctx, in.IntegrationID)
	if err != nil {
		return domain.CommunityGrant{}, err
	}
	orderID := "manual_" + hashShort(in.UserID+":"+in.ProductID+":"+in.IntegrationID+":"+in.Reason)
	requestHash := hashJSON(map[string]any{"op": "manual_grant", "user_id": in.UserID, "product_id": in.ProductID, "integration_id": in.IntegrationID, "reason": in.Reason, "tier": in.Tier})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CommunityGrant{}, err
	} else if ok {
		var out domain.CommunityGrant
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CommunityGrant{}, err
	}
	if _, err := s.mappings.FindByProductIntegration(ctx, in.ProductID, in.IntegrationID); err != nil {
		_ = s.mappings.Create(ctx, domain.ProductCommunityMapping{MappingID: "map_" + uuid.NewString(), ProductID: in.ProductID, IntegrationID: in.IntegrationID, Tier: in.Tier, RoleConfig: map[string]string{"source": "manual_grant_default"}, Enabled: true, CreatedAt: s.nowFn()})
	}
	if existing, err := s.grants.FindByOrderIntegration(ctx, orderID, in.IntegrationID); err == nil {
		_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, existing)
		return existing, nil
	}
	now := s.nowFn()
	grant := domain.CommunityGrant{GrantID: "grant_" + uuid.NewString(), UserID: in.UserID, ProductID: in.ProductID, IntegrationID: in.IntegrationID, OrderID: orderID, Tier: in.Tier, Status: domain.GrantStatusActive, GrantedAt: now, CreatedAt: now, UpdatedAt: now}
	if err := s.grants.Create(ctx, grant); err != nil {
		return domain.CommunityGrant{}, err
	}
	_ = s.appendAudit(ctx, actor, "community.grant.manual", in.UserID, integration.IntegrationID, in.ProductID, grant.GrantID, "success", in.Reason, map[string]string{"tier": in.Tier})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, grant)
	return grant, nil
}

func (s *Service) GetGrant(ctx context.Context, actor Actor, grantID string) (domain.CommunityGrant, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CommunityGrant{}, domain.ErrUnauthorized
	}
	if !isAdminOrSupport(actor) {
		return domain.CommunityGrant{}, domain.ErrForbidden
	}
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return domain.CommunityGrant{}, domain.ErrInvalidInput
	}
	return s.grants.GetByID(ctx, grantID)
}

func (s *Service) ListAuditLogs(ctx context.Context, actor Actor, userID string, from, to *time.Time) ([]domain.CommunityAuditLog, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if !isAdminOrSupport(actor) {
		return nil, domain.ErrForbidden
	}
	if s.auditLogs == nil {
		return []domain.CommunityAuditLog{}, nil
	}
	return s.auditLogs.List(ctx, strings.TrimSpace(userID), from, to)
}

func (s *Service) GetIntegrationHealth(ctx context.Context, actor Actor, integrationID string) (domain.CommunityHealthCheck, error) {
	integration, err := s.GetIntegration(ctx, actor, integrationID)
	if err != nil {
		return domain.CommunityHealthCheck{}, err
	}
	_ = integration
	return s.healthChecks.LatestByIntegrationID(ctx, integrationID)
}

func (s *Service) FlushOutbox(context.Context) error { return nil }

func isAdmin(actor Actor) bool { return strings.ToLower(strings.TrimSpace(actor.Role)) == "admin" }
func isAdminOrSupport(actor Actor) bool {
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "support"
}
func canReadCreatorResource(actor Actor, creatorID string) bool {
	return strings.TrimSpace(actor.SubjectID) == strings.TrimSpace(creatorID) || isAdminOrSupport(actor)
}

func (s *Service) appendAudit(ctx context.Context, actor Actor, action, userID, integrationID, productID, grantID, outcome, reason string, meta map[string]string) error {
	if s.auditLogs == nil {
		return nil
	}
	return s.auditLogs.Append(ctx, domain.CommunityAuditLog{AuditLogID: "audit_" + uuid.NewString(), Timestamp: s.nowFn(), ActionType: action, UserID: userID, PerformedBy: strings.TrimSpace(actor.SubjectID), PerformerRole: strings.ToLower(strings.TrimSpace(actor.Role)), IntegrationID: integrationID, ProductID: productID, GrantID: grantID, Reason: reason, Outcome: outcome, Metadata: sanitizeMap(meta)})
}

func sanitizeMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func hashShort(v string) string {
	h := hashJSON(map[string]string{"v": v})
	if len(h) > 12 {
		return h[:12]
	}
	return h
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
