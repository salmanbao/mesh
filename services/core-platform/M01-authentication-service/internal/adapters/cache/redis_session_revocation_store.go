package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisSessionRevocationStore stores revoked-session flags with TTL.
type RedisSessionRevocationStore struct {
	client *redis.Client
}

// NewRedisSessionRevocationStore creates session revocation cache adapter.
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
