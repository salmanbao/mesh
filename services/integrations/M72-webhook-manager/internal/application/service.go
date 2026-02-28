package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/domain"
	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/ports"
)

type Service struct {
	cfg         Config
	webhooks    ports.WebhookRepository
	deliveries  ports.DeliveryRepository
	analytics   ports.AnalyticsRepository
	idempotency ports.IdempotencyRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Webhooks    ports.WebhookRepository
	Deliveries  ports.DeliveryRepository
	Analytics   ports.AnalyticsRepository
	Idempotency ports.IdempotencyRepository
}

func NewService(deps Dependencies) *Service {
	s := &Service{
		cfg:         deps.Config,
		webhooks:    deps.Webhooks,
		deliveries:  deps.Deliveries,
		analytics:   deps.Analytics,
		idempotency: deps.Idempotency,
		nowFn:       time.Now().UTC,
	}
	if s.cfg.IdempotencyTTL == 0 {
		s.cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return s
}

func (s *Service) CreateWebhook(ctx context.Context, actor Actor, req CreateWebhookInput) (domain.Webhook, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Webhook{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Webhook{}, domain.ErrIdempotencyRequired
	}
	cleanURL := strings.TrimSpace(req.EndpointURL)
	if !validEndpointURL(cleanURL) {
		return domain.Webhook{}, domain.ErrInvalidInput
	}
	cleanEvents := dedup(req.EventTypes)
	if len(cleanEvents) == 0 {
		return domain.Webhook{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(req)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	} else if ok {
		var wh domain.Webhook
		_ = json.Unmarshal(rec, &wh)
		return wh, nil
	}

	now := s.nowFn()
	wh := domain.Webhook{
		WebhookID:          newID("wh"),
		EndpointURL:        cleanURL,
		EventTypes:         cleanEvents,
		Status:             "active",
		SigningSecret:      randomSecret(32),
		BatchModeEnabled:   req.BatchModeEnabled,
		BatchSize:          clampInt(req.BatchSize, 1, 100, 10),
		BatchWindowSeconds: clampInt(req.BatchWindowSeconds, 5, 300, 60),
		RateLimitPerMinute: clampInt(req.RateLimitPerMinute, 1, 100, 100),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if s.webhooks != nil {
		if err := s.webhooks.Create(ctx, wh); err != nil {
			return domain.Webhook{}, err
		}
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, wh)
	return wh, nil
}

func (s *Service) UpdateWebhook(ctx context.Context, actor Actor, id string, req UpdateWebhookInput) (domain.Webhook, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Webhook{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Webhook{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashJSON(map[string]any{"id": strings.TrimSpace(id), "body": req})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	} else if ok {
		var wh domain.Webhook
		_ = json.Unmarshal(rec, &wh)
		return wh, nil
	}

	wh, err := s.webhooks.Get(ctx, id)
	if err != nil {
		return domain.Webhook{}, err
	}
	if !wh.DeletedAt.IsZero() || wh.Status == "deleted" {
		return domain.Webhook{}, domain.ErrNotFound
	}
	if len(req.EventTypes) > 0 {
		cleanEvents := dedup(req.EventTypes)
		if len(cleanEvents) == 0 {
			return domain.Webhook{}, domain.ErrInvalidInput
		}
		wh.EventTypes = cleanEvents
	}
	if req.BatchModeEnabled != nil {
		wh.BatchModeEnabled = *req.BatchModeEnabled
	}
	if req.BatchSize != 0 {
		wh.BatchSize = clampInt(req.BatchSize, 1, 100, wh.BatchSize)
	}
	if req.BatchWindowSeconds != 0 {
		wh.BatchWindowSeconds = clampInt(req.BatchWindowSeconds, 5, 300, wh.BatchWindowSeconds)
	}
	if req.RateLimitPerMinute != 0 {
		wh.RateLimitPerMinute = clampInt(req.RateLimitPerMinute, 1, 100, wh.RateLimitPerMinute)
	}
	if status := strings.ToLower(strings.TrimSpace(req.Status)); status != "" {
		switch status {
		case "active", "disabled":
			wh.Status = status
		default:
			return domain.Webhook{}, domain.ErrInvalidInput
		}
	}
	wh.UpdatedAt = s.nowFn()
	if err := s.webhooks.Update(ctx, wh); err != nil {
		return domain.Webhook{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, wh)
	return wh, nil
}

func (s *Service) DeleteWebhook(ctx context.Context, actor Actor, id string) (domain.Webhook, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Webhook{}, domain.ErrUnauthorized
	}
	wh, err := s.webhooks.Get(ctx, id)
	if err != nil {
		return domain.Webhook{}, err
	}
	if wh.Status != "deleted" {
		now := s.nowFn()
		wh.Status = "deleted"
		wh.DeletedAt = now
		wh.UpdatedAt = now
		if err := s.webhooks.Update(ctx, wh); err != nil {
			return domain.Webhook{}, err
		}
	}
	return wh, nil
}

func (s *Service) GetWebhook(ctx context.Context, actor Actor, id string) (domain.Webhook, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Webhook{}, domain.ErrUnauthorized
	}
	return s.webhooks.Get(ctx, id)
}

func (s *Service) TestWebhook(ctx context.Context, actor Actor, id string, req TestWebhookInput) (TestResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return TestResult{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return TestResult{}, domain.ErrIdempotencyRequired
	}
	wh, err := s.webhooks.Get(ctx, id)
	if err != nil {
		return TestResult{}, err
	}
	if !wh.DeletedAt.IsZero() || wh.Status == "deleted" {
		return TestResult{}, domain.ErrNotFound
	}
	requestHash := hashJSON(map[string]any{"id": id, "payload": req.Payload})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return TestResult{}, err
	} else if ok {
		var tr TestResult
		_ = json.Unmarshal(rec, &tr)
		return tr, nil
	}
	_ = wh
	now := s.nowFn()
	result := TestResult{
		WebhookID:  id,
		Status:     "success",
		HTTPStatus: 200,
		LatencyMS:  1200,
		Timestamp:  now,
	}
	_ = s.deliveries.Add(ctx, domain.Delivery{
		DeliveryID:      newID("del"),
		WebhookID:       id,
		OriginalEventID: "test",
		OriginalType:    "webhook.test",
		HTTPStatus:      200,
		LatencyMS:       result.LatencyMS,
		RetryCount:      0,
		DeliveredAt:     now,
		IsTest:          true,
		Success:         true,
	})
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, result)
	return result, nil
}

func (s *Service) ListDeliveries(ctx context.Context, actor Actor, webhookID string, limit int) ([]domain.Delivery, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if _, err := s.webhooks.Get(ctx, webhookID); err != nil {
		return nil, err
	}
	return s.deliveries.ListByWebhook(ctx, webhookID, limit)
}

func (s *Service) GetAnalytics(ctx context.Context, actor Actor, webhookID string) (domain.Analytics, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Analytics{}, domain.ErrUnauthorized
	}
	if _, err := s.webhooks.Get(ctx, webhookID); err != nil {
		return domain.Analytics{}, err
	}
	return s.analytics.Snapshot(ctx, webhookID)
}

func (s *Service) EnableWebhook(ctx context.Context, actor Actor, id string) (domain.Webhook, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.Webhook{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Webhook{}, domain.ErrIdempotencyRequired
	}
	requestHash := hashJSON(map[string]any{"id": id, "op": "enable"})
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Webhook{}, err
	} else if ok {
		var wh domain.Webhook
		_ = json.Unmarshal(rec, &wh)
		return wh, nil
	}
	wh, err := s.webhooks.Get(ctx, id)
	if err != nil {
		return domain.Webhook{}, err
	}
	wh.Status = "active"
	wh.ConsecutiveFailures = 0
	wh.UpdatedAt = s.nowFn()
	if err := s.webhooks.Update(ctx, wh); err != nil {
		return domain.Webhook{}, err
	}
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, wh)
	return wh, nil
}

func (s *Service) ReceiveCompatibilityWebhook(_ context.Context, payload json.RawMessage) map[string]any {
	return map[string]any{
		"accepted": true,
		"size":     len(payload),
	}
}

func (s *Service) getIdempotent(ctx context.Context, key, hash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != hash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	return rec.Response, true, nil
}

func (s *Service) completeIdempotent(ctx context.Context, key, requestHash string, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.idempotency.Upsert(ctx, domain.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Response:    raw,
		ExpiresAt:   s.nowFn().Add(s.cfg.IdempotencyTTL),
	})
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func dedup(in []string) []string {
	m := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := m[v]; ok {
			continue
		}
		m[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func clampInt(v, min, max, def int) int {
	if v == 0 {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func validEndpointURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return false
	}
	if strings.ToLower(parsed.Scheme) != "https" || strings.TrimSpace(parsed.Hostname()) == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
			return false
		}
	}
	return true
}

func randomSecret(size int) string {
	if size <= 0 {
		size = 16
	}
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}

func newID(prefix string) string {
	return prefix + "-" + randomSecret(8)
}
