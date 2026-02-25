package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// RedisMFAChallengeStore stores temporary MFA challenges in Redis.
type RedisMFAChallengeStore struct {
	client *redis.Client
}

// NewRedisMFAChallengeStore creates MFA challenge cache adapter.
func NewRedisMFAChallengeStore(client *redis.Client) *RedisMFAChallengeStore {
	return &RedisMFAChallengeStore{client: client}
}

func (s *RedisMFAChallengeStore) Put(ctx context.Context, token string, challenge ports.MFAChallenge, ttl time.Duration) error {
	raw, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "auth:mfa:"+token, raw, ttl).Err()
}

func (s *RedisMFAChallengeStore) Get(ctx context.Context, token string) (*ports.MFAChallenge, error) {
	raw, err := s.client.Get(ctx, "auth:mfa:"+token).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	var out ports.MFAChallenge
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *RedisMFAChallengeStore) Delete(ctx context.Context, token string) error {
	return s.client.Del(ctx, "auth:mfa:"+token).Err()
}
