package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
)

func Connect(_ context.Context, redisURL string) (*redis.Client, error) {
	if strings.HasPrefix(redisURL, "redis://") {
		opt, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			return nil, fmt.Errorf("parse redis url: %w", parseErr)
		}
		return redis.NewClient(opt), nil
	}
	return redis.NewClient(&redis.Options{Addr: redisURL}), nil
}

type RedisLockoutStore struct {
	client *redis.Client
}

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
			p.Expire(ctx, redisKey, lockoutWindow+30*time.Minute)
			return nil
		})
		if err != nil {
			return ports.LockoutState{}, err
		}
		state.LockedUntil = &lockedUntil
		return state, nil
	}

	_ = s.client.Expire(ctx, redisKey, 24*time.Hour).Err()
	return state, nil
}

func (s *RedisLockoutStore) Clear(ctx context.Context, key string) error {
	return s.client.Del(ctx, "auth:lockout:"+key).Err()
}

type RedisSessionRevocationStore struct {
	client *redis.Client
}

func NewRedisSessionRevocationStore(client *redis.Client) *RedisSessionRevocationStore {
	return &RedisSessionRevocationStore{client: client}
}

func (s *RedisSessionRevocationStore) MarkRevoked(ctx context.Context, sessionID uuid.UUID, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Hour
	}
	return s.client.Set(ctx, "auth:revoked:"+sessionID.String(), "1", ttl).Err()
}

func (s *RedisSessionRevocationStore) IsRevoked(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	n, err := s.client.Exists(ctx, "auth:revoked:"+sessionID.String()).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

type RedisMFAChallengeStore struct {
	client *redis.Client
}

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

type RedisOIDCStateStore struct {
	client *redis.Client
}

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
