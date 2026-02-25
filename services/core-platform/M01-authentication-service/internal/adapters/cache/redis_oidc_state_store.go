package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// RedisOIDCStateStore stores short-lived OIDC state/PKCE envelopes.
type RedisOIDCStateStore struct {
	client *redis.Client
}

// NewRedisOIDCStateStore creates OIDC state cache adapter.
func NewRedisOIDCStateStore(client *redis.Client) *RedisOIDCStateStore {
	return &RedisOIDCStateStore{client: client}
}

func (s *RedisOIDCStateStore) Put(ctx context.Context, state string, value ports.OIDCAuthState, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "auth:oidc:state:"+state, raw, ttl).Err()
}

func (s *RedisOIDCStateStore) Get(ctx context.Context, state string) (*ports.OIDCAuthState, error) {
	raw, err := s.client.Get(ctx, "auth:oidc:state:"+state).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	var out ports.OIDCAuthState
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *RedisOIDCStateStore) Delete(ctx context.Context, state string) error {
	return s.client.Del(ctx, "auth:oidc:state:"+state).Err()
}
