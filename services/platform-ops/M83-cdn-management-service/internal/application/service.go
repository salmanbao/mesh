package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/domain"
	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/ports"
)

type Service struct {
	cfg          Config
	configs      ports.ConfigRepository
	purges       ports.PurgeRepository
	metrics      ports.MetricsRepository
	certificates ports.CertificateRepository
	idempotency  ports.IdempotencyRepository
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	Configs      ports.ConfigRepository
	Purges       ports.PurgeRepository
	Metrics      ports.MetricsRepository
	Certificates ports.CertificateRepository
	Idempotency  ports.IdempotencyRepository
}

var idCounter uint64

func NewService(deps Dependencies) *Service {
	s := &Service{
		cfg:          deps.Config,
		configs:      deps.Configs,
		purges:       deps.Purges,
		metrics:      deps.Metrics,
		certificates: deps.Certificates,
		idempotency:  deps.Idempotency,
		nowFn:        time.Now().UTC,
	}
	if s.cfg.IdempotencyTTL == 0 {
		s.cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	return s
}

func (s *Service) CreateConfig(ctx context.Context, actor Actor, input CreateConfigInput) (domain.CDNConfig, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CDNConfig{}, domain.ErrUnauthorized
	}
	if !isAdmin(actor.Role) {
		return domain.CDNConfig{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CDNConfig{}, domain.ErrIdempotencyRequired
	}
	provider := normalizeProvider(input.Provider)
	if provider == "" || len(input.Config) == 0 {
		return domain.CDNConfig{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(input)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CDNConfig{}, err
	} else if ok {
		var config domain.CDNConfig
		_ = json.Unmarshal(rec, &config)
		return config, nil
	}
	version := 1
	if latest, err := s.configs.Latest(ctx); err == nil {
		version = latest.Version + 1
	}
	now := s.nowFn()
	config := domain.CDNConfig{
		ConfigID:     newID("cfg", now),
		Provider:     provider,
		Version:      version,
		Status:       "active",
		Config:       cloneMap(input.Config),
		TLSVersion:   "TLS1.3",
		HeaderRules:  cloneStringMap(input.HeaderRules),
		SignedURLTTL: clampInt(input.SignedURLTTL, 60, 86400, 3600),
		CreatedBy:    actor.SubjectID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.configs.Create(ctx, config); err != nil {
		return domain.CDNConfig{}, err
	}
	metrics, _ := s.metrics.Snapshot(ctx)
	metrics.ActiveConfigVer = config.Version
	_ = s.metrics.SetSnapshot(ctx, metrics)
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, config)
	return config, nil
}

func (s *Service) ListConfigs(ctx context.Context, actor Actor) ([]domain.CDNConfig, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.configs.List(ctx)
}

func (s *Service) Purge(ctx context.Context, actor Actor, input PurgeInput) (domain.PurgeRequest, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.PurgeRequest{}, domain.ErrUnauthorized
	}
	if !isAdmin(actor.Role) {
		return domain.PurgeRequest{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.PurgeRequest{}, domain.ErrIdempotencyRequired
	}
	scope := normalizeScope(input.Scope)
	target := strings.TrimSpace(input.Target)
	if scope == "" || target == "" {
		return domain.PurgeRequest{}, domain.ErrInvalidInput
	}
	requestHash := hashJSON(input)
	if rec, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.PurgeRequest{}, err
	} else if ok {
		var purge domain.PurgeRequest
		_ = json.Unmarshal(rec, &purge)
		return purge, nil
	}
	now := s.nowFn()
	purge := domain.PurgeRequest{
		PurgeID:     newID("purge", now),
		Scope:       scope,
		Target:      target,
		Status:      "completed",
		CompletedIn: 15,
		RequestedBy: actor.SubjectID,
		CreatedAt:   now,
	}
	if err := s.purges.Create(ctx, purge); err != nil {
		return domain.PurgeRequest{}, err
	}
	metrics, _ := s.metrics.Snapshot(ctx)
	metrics.PendingPurges = 0
	_ = s.metrics.SetSnapshot(ctx, metrics)
	_ = s.completeIdempotent(ctx, actor.IdempotencyKey, requestHash, purge)
	return purge, nil
}

func (s *Service) Metrics(ctx context.Context) (map[string]any, error) {
	metrics, err := s.metrics.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	certs, err := s.certificates.List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"summary":      metrics,
		"certificates": certs,
	}, nil
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
	return s.idempotency.Upsert(ctx, domain.IdempotencyRecord{Key: key, RequestHash: requestHash, Response: raw, ExpiresAt: s.nowFn().Add(s.cfg.IdempotencyTTL)})
}

func normalizeProvider(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "cloudflare":
		return "cloudflare"
	case "cloudfront":
		return "cloudfront"
	default:
		return ""
	}
}

func normalizeScope(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "url", "prefix", "tag":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func isAdmin(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == "admin" || role == "ops_admin" || role == "support_manager"
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

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func newID(prefix string, now time.Time) string {
	n := atomic.AddUint64(&idCounter, 1)
	return prefix + "-" + shortID(now.UnixNano()+int64(n))
}

func shortID(v int64) string {
	if v < 0 {
		v = -v
	}
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 16)
	for v > 0 {
		buf = append(buf, chars[v%int64(len(chars))])
		v /= int64(len(chars))
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
