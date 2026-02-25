package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// RedisRegistrationCompletionStore stores short-lived registration completion tokens.
type RedisRegistrationCompletionStore struct {
	client *redis.Client
}

// NewRedisRegistrationCompletionStore creates completion-token cache adapter.
func NewRedisRegistrationCompletionStore(client *redis.Client) *RedisRegistrationCompletionStore {
	return &RedisRegistrationCompletionStore{client: client}
}

func (s *RedisRegistrationCompletionStore) Put(ctx context.Context, token string, value ports.RegistrationCompletion, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "auth:register:complete:"+token, raw, ttl).Err()
}

func (s *RedisRegistrationCompletionStore) Get(ctx context.Context, token string) (*ports.RegistrationCompletion, error) {
	raw, err := s.client.Get(ctx, "auth:register:complete:"+token).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	var out ports.RegistrationCompletion
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *RedisRegistrationCompletionStore) Delete(ctx context.Context, token string) error {
	return s.client.Del(ctx, "auth:register:complete:"+token).Err()
}
