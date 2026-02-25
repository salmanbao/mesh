package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

// RedisLockoutStore implements brute-force lockout storage in Redis.
type RedisLockoutStore struct {
	client *redis.Client
}

// NewRedisLockoutStore creates a lockout store backed by Redis hashes.
func NewRedisLockoutStore(client *redis.Client) *RedisLockoutStore {
	return &RedisLockoutStore{client: client}
}

func (s *RedisLockoutStore) Get(ctx context.Context, key string) (ports.LockoutState, error) {
	data, err := s.client.HGetAll(ctx, "auth:lockout:"+key).Result()
	if err != nil {
		return ports.LockoutState{}, err
	}
	if len(data) == 0 {
		return ports.LockoutState{}, nil
	}

	state := ports.LockoutState{}
	if raw, ok := data["failed_count"]; ok {
		if n, convErr := strconv.Atoi(raw); convErr == nil {
			state.FailedCount = n
		}
	}
	if raw, ok := data["locked_until"]; ok && raw != "" {
		if unix, convErr := strconv.ParseInt(raw, 10, 64); convErr == nil && unix > 0 {
			t := time.Unix(unix, 0).UTC()
			state.LockedUntil = &t
		}
	}
	return state, nil
}

func (s *RedisLockoutStore) RecordFailure(ctx context.Context, key string, now time.Time, threshold int, lockoutWindow time.Duration) (ports.LockoutState, error) {
	redisKey := "auth:lockout:" + key

	count, err := s.client.HIncrBy(ctx, redisKey, "failed_count", 1).Result()
	if err != nil {
		return ports.LockoutState{}, err
	}

	state := ports.LockoutState{FailedCount: int(count)}
	if int(count) >= threshold {
		lockedUntil := now.Add(lockoutWindow).UTC()
		_, err = s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.HSet(ctx, redisKey, "locked_until", lockedUntil.Unix())
			p.Expire(ctx, redisKey, lockoutWindow+30*time.Minute) // reasonable TTL to auto-clear stale lockouts, with buffer for manual clears
			return nil
		})
		if err != nil {
			return ports.LockoutState{}, err
		}
		state.LockedUntil = &lockedUntil
		return state, nil
	}

	_ = s.client.Expire(ctx, redisKey, 24*time.Hour).Err() // reasonable TTL to auto-clear stale lockouts
	return state, nil
}

func (s *RedisLockoutStore) Clear(ctx context.Context, key string) error {
	return s.client.Del(ctx, "auth:lockout:"+key).Err()
}
