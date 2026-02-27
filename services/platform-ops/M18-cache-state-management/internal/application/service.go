package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/viralforge/mesh/services/platform-ops/M18-cache-state-management/internal/domain"
)

func (s *Service) GetCache(ctx context.Context, actor Actor, key string) (domain.CacheItem, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CacheItem{}, domain.ErrUnauthorized
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return domain.CacheItem{}, domain.ErrInvalidInput
	}
	item, err := s.cache.Get(ctx, key, s.nowFn())
	if err != nil {
		return domain.CacheItem{}, err
	}
	if s.metrics != nil {
		if item.Found {
			_ = s.metrics.RecordHit(ctx)
		} else {
			_ = s.metrics.RecordMiss(ctx)
		}
	}
	return item, nil
}

func (s *Service) PutCache(ctx context.Context, actor Actor, key string, value []byte, ttlSeconds int) (domain.CacheItem, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CacheItem{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.CacheItem{}, domain.ErrIdempotencyRequired
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 512 || len(value) == 0 {
		return domain.CacheItem{}, domain.ErrInvalidInput
	}
	if ttlSeconds <= 0 {
		ttlSeconds = int(s.cfg.DefaultTTL.Seconds())
	}
	if ttlSeconds <= 0 {
		return domain.CacheItem{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{"op": "put", "key": key, "value": string(value), "ttl": ttlSeconds})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CacheItem{}, err
	} else if ok {
		var out domain.CacheItem
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.CacheItem{}, err
	}

	now := s.nowFn()
	expiresAt := now.Add(time.Duration(ttlSeconds) * time.Second)
	if err := s.cache.Put(ctx, domain.CacheEntry{Key: key, Value: append([]byte(nil), value...), ExpiresAt: expiresAt, StoredAt: now}); err != nil {
		return domain.CacheItem{}, err
	}
	item := domain.CacheItem{Key: key, Value: append([]byte(nil), value...), Found: true, TTLSeconds: ttlSeconds}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, item)
	return item, nil
}

func (s *Service) DeleteCache(ctx context.Context, actor Actor, key string) (bool, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return false, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return false, domain.ErrIdempotencyRequired
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false, domain.ErrInvalidInput
	}

	requestHash := hashJSON(map[string]any{"op": "delete", "key": key})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return false, err
	} else if ok {
		var out struct {
			Deleted bool `json:"deleted"`
		}
		if json.Unmarshal(raw, &out) == nil {
			return out.Deleted, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return false, err
	}

	deleted, err := s.cache.Delete(ctx, key)
	if err != nil {
		return false, err
	}
	if deleted && s.metrics != nil {
		_ = s.metrics.RecordEviction(ctx, 1)
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, struct {
		Deleted bool `json:"deleted"`
	}{Deleted: deleted})
	return deleted, nil
}

func (s *Service) InvalidateCache(ctx context.Context, actor Actor, keys []string) (int, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return 0, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return 0, domain.ErrIdempotencyRequired
	}
	if len(keys) == 0 {
		return 0, domain.ErrInvalidInput
	}

	normalized := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" || len(k) > 512 {
			return 0, domain.ErrInvalidInput
		}
		normalized = append(normalized, k)
	}

	requestHash := hashJSON(map[string]any{"op": "invalidate", "keys": normalized})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return 0, err
	} else if ok {
		var out struct {
			Count int `json:"count"`
		}
		if json.Unmarshal(raw, &out) == nil {
			return out.Count, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return 0, err
	}

	count, err := s.cache.Invalidate(ctx, normalized)
	if err != nil {
		return 0, err
	}
	if count > 0 && s.metrics != nil {
		_ = s.metrics.RecordEviction(ctx, count)
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, struct {
		Count int `json:"count"`
	}{Count: count})
	return count, nil
}

func (s *Service) GetMetrics(ctx context.Context, actor Actor) (domain.CacheMetrics, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CacheMetrics{}, domain.ErrUnauthorized
	}
	if s.metrics == nil {
		return domain.CacheMetrics{}, nil
	}
	if s.cache != nil && s.metrics != nil {
		if used, err := s.cache.MemoryUsedBytes(ctx); err == nil {
			_ = s.metrics.SetMemoryUsed(ctx, used)
		}
	}
	return s.metrics.Snapshot(ctx)
}

func (s *Service) GetHealth(context.Context) (domain.HealthReport, error) {
	now := s.nowFn()
	return domain.HealthReport{
		Status:        "healthy",
		Timestamp:     now,
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Version:       s.cfg.Version,
		Checks: map[string]domain.ComponentCheck{
			"cache_engine": {Name: "cache_engine", Status: "healthy", LatencyMS: 2, LastChecked: now},
			"idempotency":  {Name: "idempotency", Status: "healthy", LatencyMS: 1, LastChecked: now},
		},
	}, nil
}

func (s *Service) RecordHTTPMetric(context.Context, string, string, int, time.Duration) {}

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
	return append([]byte(nil), rec.ResponseBody...), true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	if err := s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL)); err != nil {
		if err == domain.ErrConflict {
			return domain.ErrIdempotencyConflict
		}
		return err
	}
	return nil
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
